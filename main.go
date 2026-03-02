package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"

	colorBold = "\033[1m"
	colorDim  = "\033[2m"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type GPULogEntry struct {
	Timestamp   string             `json:"timestamp"`
	GPUUsage    float64            `json:"gpu_usage_percent"`
	MemoryUsage int                `json:"memory_usage_mb"`
	Frequencies map[string]float64 `json:"frequencies"`
}

type GPUMonitor struct {
	gpuCores       int
	maxMemory      int
	chipName       string
	historyUsage   []float64
	historyMemory  []int
	gpuFrequencies map[string]float64
}

func NewGPUMonitor() *GPUMonitor {
	monitor := &GPUMonitor{
		historyUsage:   make([]float64, 0, 60),
		historyMemory:  make([]int, 0, 60),
		gpuFrequencies: make(map[string]float64),
	}
	monitor.getGPUInfo()
	return monitor
}

func (m *GPUMonitor) logGPUUsage(usage float64, memory int) {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	now := time.Now()
	logFileName := filepath.Join(logDir, fmt.Sprintf("gpu_monitor_%s.log", now.Format("2006-01-02")))

	entry := GPULogEntry{
		Timestamp:   now.Format(time.RFC3339),
		GPUUsage:    usage,
		MemoryUsage: memory,
		Frequencies: make(map[string]float64),
	}

	for k, v := range m.gpuFrequencies {
		entry.Frequencies[k] = v
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return
	}

	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return
	}
}

func (m *GPUMonitor) getGPUInfo() {
	cmd := exec.Command("system_profiler", "SPHardwareDataType")
	result, err := cmd.CombinedOutput()
	if err != nil {
		m.gpuCores = 8
		m.maxMemory = 16384
		m.chipName = "未知"
		return
	}

	output := string(result)

	chipRe := regexp.MustCompile(`Chip:\s+Apple\s+(M[1-3])`)
	chipMatch := chipRe.FindStringSubmatch(output)
	if len(chipMatch) > 1 {
		m.chipName = chipMatch[1]

		if strings.Contains(m.chipName, "M1") {
			if strings.Contains(m.chipName, "Max") || strings.Contains(m.chipName, "Pro") {
				m.gpuCores = 7
				m.maxMemory = 8192
			} else {
				m.gpuCores = 8
				m.maxMemory = 16384
			}
		} else if strings.Contains(m.chipName, "M2") {
			if strings.Contains(m.chipName, "Max") || strings.Contains(m.chipName, "Pro") {
				m.gpuCores = 10
				m.maxMemory = 10240
			} else {
				m.gpuCores = 10
				m.maxMemory = 24576
			}
		} else if strings.Contains(m.chipName, "M3") {
			if strings.Contains(m.chipName, "Max") || strings.Contains(m.chipName, "Pro") {
				m.gpuCores = 10
				m.maxMemory = 14336
			} else {
				m.gpuCores = 10
				m.maxMemory = 36864
			}
		} else {
			m.gpuCores = 8
			m.maxMemory = 16384
		}
	} else {
		m.gpuCores = 8
		m.maxMemory = 16384
		m.chipName = "未知"
	}
}

func (m *GPUMonitor) getGPUUsage() (float64, int) {
	cmd := exec.Command("powermetrics", "--samplers", "all", "-i", "1000", "-n", "1")
	result, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\n无法获取GPU使用信息: %v\n", err)
		fmt.Println("请使用 sudo 权限运行程序")
		return 0.0, 0
	}

	output := string(result)

	usage := 0.0
	memory := 0

	activeRe := regexp.MustCompile(`GPU\s+HW\s+active\s+residency:\s+(\d+\.?\d*)%`)
	activeMatch := activeRe.FindStringSubmatch(output)
	if len(activeMatch) > 1 {
		fmt.Sscanf(activeMatch[1], "%f", &usage)
	}

	if usage == 0.0 {
		idleRe := regexp.MustCompile(`GPU\s+idle\s+residency:\s+(\d+\.?\d*)%`)
		idleMatch := idleRe.FindStringSubmatch(output)
		if len(idleMatch) > 1 {
			idle := 0.0
			fmt.Sscanf(idleMatch[1], "%f", &idle)
			usage = 100.0 - idle
			if usage < 0 {
				usage = 0.0
			}
		}
	}

	freqRe := regexp.MustCompile(`GPU\s+HW\s+active\s+residency:\s+\d+\.?\d*%\s*\(([^)]+)\)`)
	freqMatch := freqRe.FindStringSubmatch(output)
	if len(freqMatch) > 1 {
		m.gpuFrequencies = make(map[string]float64)
		freqStr := freqMatch[1]

		freqParts := strings.Fields(freqStr)
		for i := 0; i < len(freqParts); i += 3 {
			if i+2 < len(freqParts) {
				freq := freqParts[i]
				percentStr := strings.TrimSuffix(freqParts[i+2], "%")
				percent := 0.0
				fmt.Sscanf(percentStr, "%f", &percent)
				if percent > 0 {
					m.gpuFrequencies[freq] = percent
				}
			}
		}
	}

	if usage > 100.0 {
		usage = 100.0
	}

	return usage, memory
}

func (m *GPUMonitor) updateHistory(usage float64, memory int) {
	m.historyUsage = append(m.historyUsage, usage)
	if len(m.historyUsage) > 60 {
		m.historyUsage = m.historyUsage[1:]
	}

	m.historyMemory = append(m.historyMemory, memory)
	if len(m.historyMemory) > 60 {
		m.historyMemory = m.historyMemory[1:]
	}
}

func (m *GPUMonitor) display(usage float64, memory int) {
	fmt.Print("\033[H\033[2J")
	fmt.Print("\033[3J")

	fmt.Printf("%s╔%s═══════════════════════════════════════╗%s\n", colorCyan, colorCyan, colorReset)
	fmt.Printf("%s║%s                    Mac GPU Monitor                      %s║%s\n", colorCyan, colorBold+colorCyan, colorReset, colorCyan)
	fmt.Printf("%s╚%s═══════════════════════════════════════════╝%s\n", colorCyan, colorCyan, colorReset)
	fmt.Println()

	fmt.Printf("%s┌%s──────────────────────────────────────────────────┐%s\n", colorCyan, colorCyan, colorReset)
	fmt.Printf("%s│%s %s🍎%s 芯片型号: %s%-16s%s │%s\n", colorCyan, colorReset, colorYellow, colorWhite, colorBold, m.chipName, colorReset, colorCyan)
	fmt.Printf("%s│%s %s💻%s 核心数量: %s%-16d%s │%s\n", colorCyan, colorReset, colorYellow, colorWhite, colorBold, m.gpuCores, colorReset, colorCyan)
	fmt.Printf("%s│%s %s💾%s 显存总量: %s%-16d MB%s   │%s\n", colorCyan, colorReset, colorYellow, colorWhite, colorBold, m.maxMemory, colorReset, colorCyan)
	fmt.Printf("%s└%s──────────────────────────────────────────────────┘%s\n", colorCyan, colorCyan, colorReset)
	fmt.Println()

	fmt.Printf("%s┌%s──────────────────────────────────────────────────┐%s\n", colorCyan, colorCyan, colorReset)
	fmt.Printf("%s│%s %s📊%s GPU 使用率                                 │%s\n", colorCyan, colorReset, colorYellow, colorReset, colorCyan)
	fmt.Printf("%s│%s                                                │%s\n", colorCyan, colorReset, colorCyan)
	fmt.Printf("%s│%s  ", colorCyan, colorReset)

	barWidth := 40
	filledWidth := int(usage / 100.0 * float64(barWidth))
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			if usage < 30 {
				fmt.Print(colorGreen + "█" + colorReset)
			} else if usage < 70 {
				fmt.Print(colorYellow + "█" + colorReset)
			} else {
				fmt.Print(colorRed + "█" + colorReset)
			}
		} else {
			fmt.Print(colorDim + "░" + colorReset)
		}
	}
	fmt.Printf("  %s%.1f%%%s                                     │%s\n", colorBold, usage, colorReset, colorCyan)
	fmt.Printf("%s└%s──────────────────────────────────────────────────┘%s\n", colorCyan, colorCyan, colorReset)
	fmt.Println()

	if len(m.gpuFrequencies) > 0 {
		fmt.Printf("%s┌%s──────────────────────────────────────────────────┐%s\n", colorCyan, colorCyan, colorReset)
		fmt.Printf("%s│%s %s📈%s GPU 频率分布                             │%s\n", colorCyan, colorReset, colorYellow, colorReset, colorCyan)
		fmt.Printf("%s│%s %s(显示 GPU 在不同频率下的工作时间占比)%s               │%s\n", colorCyan, colorReset, colorDim, colorReset, colorCyan)
		fmt.Printf("%s│%s                                                │%s\n", colorCyan, colorReset, colorCyan)
		fmt.Println()

		sortedFreqs := make([]string, 0, len(m.gpuFrequencies))
		for freq := range m.gpuFrequencies {
			sortedFreqs = append(sortedFreqs, freq)
		}

		for i := 0; i < len(sortedFreqs); i++ {
			for j := i + 1; j < len(sortedFreqs); j++ {
				freqI, _ := strconv.Atoi(sortedFreqs[i])
				freqJ, _ := strconv.Atoi(sortedFreqs[j])
				if freqI > freqJ {
					sortedFreqs[i], sortedFreqs[j] = sortedFreqs[j], sortedFreqs[i]
				}
			}
		}

		for _, freq := range sortedFreqs {
			percent := m.gpuFrequencies[freq]
			if percent > 0.1 {
				fmt.Printf("%s│%s  %s%-8s%s: ", colorCyan, colorReset, colorDim, freq+" MHz", colorReset)
				barFilled := int(percent / 100.0 * float64(barWidth-15))
				for k := 0; k < barWidth-15; k++ {
					if k < barFilled {
						fmt.Print(colorBlue + "█" + colorReset)
					} else {
						fmt.Print(colorDim + "░" + colorReset)
					}
				}
				fmt.Printf(" %s%.1f%%%s                                   │%s\n", colorBold, percent, colorReset, colorCyan)
			}
		}

		fmt.Printf("%s└%s──────────────────────────────────────────────────┘%s\n", colorCyan, colorCyan, colorReset)
		fmt.Println()
	}

	fmt.Printf("%s┌%s──────────────────────────────────────────────────┐%s\n", colorCyan, colorCyan, colorReset)
	fmt.Printf("%s│%s %s📊%s GPU 使用历史 (最近 60 秒)                     │%s\n", colorCyan, colorReset, colorYellow, colorReset, colorCyan)
	fmt.Printf("%s│%s                                                │%s\n", colorCyan, colorReset, colorCyan)
	fmt.Println()

	barHeight := 10
	for i := barHeight; i >= 0; i-- {
		threshold := float64(i) / 10.0 * 100.0
		fmt.Printf("%s│%s  %s%3.0f%%%s │", colorCyan, colorReset, colorDim, threshold, colorReset)

		for _, usage := range m.historyUsage {
			if usage >= threshold {
				fmt.Print(colorPurple + "█" + colorReset)
			} else {
				fmt.Print(colorDim + "░" + colorReset)
			}
		}

		if len(m.historyUsage) < 60 {
			fmt.Print(colorDim + strings.Repeat("░", 60-len(m.historyUsage)) + colorReset)
		}
		fmt.Printf(" %s│%s\n", colorCyan, colorReset)
	}

	fmt.Printf("%s│%s      └", colorCyan, colorReset)
	for i := 0; i < 60; i += 10 {
		if i < len(m.historyUsage) {
			fmt.Printf("%s─%s%2ds%s", colorCyan, colorCyan, i, colorReset)
		} else {
			fmt.Printf("%s─%s────%s", colorCyan, colorCyan, colorReset)
		}
	}
	if len(m.historyUsage) > 50 {
		fmt.Printf("%s─%s%2ds%s", colorCyan, colorCyan, len(m.historyUsage), colorReset)
	}
	fmt.Printf(" %s│%s\n", colorCyan, colorReset)
	fmt.Printf("%s└%s──────────────────────────────────────────────────┘%s\n", colorCyan, colorCyan, colorReset)
	fmt.Println()

	fmt.Printf("%s按 %sq%s 或 %sCtrl+C%s 退出程序%s\n", colorDim, colorReset, colorDim, colorReset, colorReset, colorReset)
}

func (m *GPUMonitor) checkSudo() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}

	if currentUser.Uid == "0" {
		return true
	}

	return false
}

func (m *GPUMonitor) run() {
	if !m.checkSudo() {
		warnLine1 := colorYellow + "╔═════════════════════════════════════════╗" + colorReset
		warnLine2 := colorYellow + "║" + colorReset + colorYellow + "  警告: 需要管理员权限才能获取 GPU 信息            " + colorReset + colorYellow + "║" + colorReset
		warnLine3 := colorYellow + "╚══════════════════════════════════════╝" + colorReset
		fmt.Println(warnLine1)
		fmt.Println(warnLine2)
		fmt.Println(warnLine3)
		fmt.Println()

		helpLine1 := colorCyan + "请使用以下命令运行:" + colorReset
		helpLine2 := colorBold + "  sudo ./mac_gpu" + colorReset
		fmt.Println(helpLine1)
		fmt.Println(helpLine2)
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("是否继续尝试运行? (y/n): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Println("程序退出")
			return
		}
	}

	ticker := time.NewTicker(1 * time.Second)

	fmt.Println("正在启动 GPU 监控...")
	time.Sleep(1 * time.Second)

	for range ticker.C {
		usage, memory := m.getGPUUsage()
		m.updateHistory(usage, memory)
		m.display(usage, memory)
		m.logGPUUsage(usage, memory)
	}
}

func main() {
	monitor := NewGPUMonitor()
	monitor.run()
}
