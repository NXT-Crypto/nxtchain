package nextutils

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	debugEnabled bool
	debugLogger  *log.Logger

	colorReset = "\033[0m"

	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

func EnableDebug(logFile string) error {
	debugEnabled = true
	if logFile == "" {
		debugLogger = log.New(os.Stdout, "", log.LstdFlags)
		return nil
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	debugLogger = log.New(file, "", log.LstdFlags)
	return nil
}

func InitDebugger(withFile bool, options ...bool) {
	debugEnabled = false
	if len(options) > 0 {
		debugEnabled = options[0]
	}
	if withFile {
		if err := os.MkdirAll("logs", 0755); err != nil {
			log.Printf("Failed to create logs directory: %v", err)
			return
		}
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		logFile := fmt.Sprintf("logs/%s.log", timestamp)
		if err := EnableDebug(logFile); err != nil {
			log.Printf("Failed to initialize debug logging: %v", err)
		}
	} else {
		debugLogger = log.New(os.Stdout, "", log.LstdFlags)
	}
}

func Debug(format string, v ...interface{}) {
	if debugEnabled && debugLogger != nil {
		msg := fmt.Sprintf(format, v...)
		debugLogger.Printf("[%sDEBUG%s] %s", colorBlue, colorReset, msg)
	}
}

func Error(format string, v ...interface{}) {
	if debugLogger != nil {
		msg := fmt.Sprintf(format, v...)
		debugLogger.Printf("[%sERROR%s] %s", colorRed, colorReset, msg)
	}
}

func Info(format string, v ...interface{}) {
	if debugLogger != nil {
		msg := fmt.Sprintf(format, v...)
		debugLogger.Printf("[%sINFO%s] %s", colorGreen, colorReset, msg)
	}
}

func NewLine() {
	if debugEnabled && debugLogger != nil {
		debugLogger.Println()
		debugLogger.Println("=============================================")
		debugLogger.Println()
	}
}

func PrintLogo(addText string, dev bool) {
	logo := fmt.Sprintf(`   _  ___  _______________ _____   _____  __
  / |/ / |/_/_  __/ ___/ // / _ | /  _/ |/ /
 /    />  <  / / / /__/ _  / __ |_/ //    / 
/_/|_/_/|_| /_/  \___/_//_/_/ |_/___/_/|_/  
%s `, addText)

	fmt.Println()
	fmt.Print(logo)

	if dev {
		devMsg := "\n+===========================================+\n|< [ D E V E L O P E M E N T  -  M O D E ] >|\n+===========================================+"
		fmt.Println()
		fmt.Println(devMsg)
	}
	fmt.Println()
}
