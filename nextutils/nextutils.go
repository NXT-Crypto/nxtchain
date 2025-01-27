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

func InitDebugger(withFile bool) {
	debugEnabled = true
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
		debugLogger.Printf("[DEBUG] %s", msg)
	}
}

func Error(format string, v ...interface{}) {
	if debugEnabled && debugLogger != nil {
		msg := fmt.Sprintf(format, v...)
		debugLogger.Printf("[ERROR] %s", msg)
	}
}

func PrintLogo(addText string, dev bool) {
	logo := fmt.Sprintf(`   _  ___  _______________ _____   _____  __
  / |/ / |/_/_  __/ ___/ // / _ | /  _/ |/ /
 /    />  <  / / / /__/ _  / __ |_/ //    / 
/_/|_/_/|_| /_/  \___/_//_/_/ |_/___/_/|_/  
%s `, addText)

	fmt.Print(logo)

	if dev {
		devMsg := "\n=============================================\n   [ D E V E L O P E M E N T    M O D E ]\n============================================="
		fmt.Println(devMsg)
	}
}
