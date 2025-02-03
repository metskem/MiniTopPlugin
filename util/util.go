package util

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/metskem/MiniTopPlugin/conf"
	"os"
	"time"
)

var logFile *os.File

// GetFormattedUnit - Transform the input (integer) to a string formatted in K, M or G */
func GetFormattedUnit(unitValue float64) string {
	if unitValue == 0 {
		return "-"
	}
	unitValueInt := int(unitValue)
	if unitValueInt >= 10*1000*1000*1000 {
		return fmt.Sprintf("%dG", unitValueInt/1000/1000/1000)
	} else if unitValueInt >= 10*1000*1000 {
		return fmt.Sprintf("%dM", unitValueInt/1000/1000)
	} else if unitValueInt >= 10*1000 {
		return fmt.Sprintf("%dK", unitValueInt/1000)
	} else {
		return fmt.Sprintf("%d", unitValueInt)
	}
}

// GetFormattedElapsedTime - Transform the input (time in nanoseconds) to a string with number of days, hours, mins and secs, like "1d01h54m10s" */
func GetFormattedElapsedTime(timeInNanoSecs float64) string {
	if timeInNanoSecs == 0 {
		return "-"
	}
	timeInSecs := int64(timeInNanoSecs / 1e9)
	days := timeInSecs / 86400
	secsLeft := timeInSecs % 86400
	hours := secsLeft / 3600
	secsLeft = secsLeft % 3600
	mins := secsLeft / 60
	secs := secsLeft % 60
	if days > 0 {
		return fmt.Sprintf("%dd%02dh%02dm%02ds", days, hours, mins, secs)
	} else if hours > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", hours, mins, secs)
	} else if mins > 0 {
		return fmt.Sprintf("%dm%02ds", mins, secs)
	} else {
		return fmt.Sprintf("%ds", secs)
	}
}

func WriteToFileDebug(text string) {
	if conf.UseDebugging {
		WriteToFile(text)
	}
}

func WriteToFile(text string) {
	var err error
	if logFile == nil {
		if logFile, err = os.OpenFile(conf.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
	}
	_, _ = logFile.WriteString(time.Now().Format(time.RFC3339) + " " + text + "\n")
}

func TruncateString(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}

func IsTokenValid(tokenString string) (isValid bool) {
	// Parse the token without verifying the signature
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		fmt.Println("Error parsing token:", err)
		isValid = false
	}
	// Extract claims
	if claims, claimOk := token.Claims.(jwt.MapClaims); claimOk {
		if exp, expOk := claims["exp"].(float64); expOk {
			expireTime := time.Unix(int64(exp), 0)
			now := time.Now()
			if now.After(expireTime) {
				isValid = false
				WriteToFileDebug(fmt.Sprintf("Token is expired: %s", expireTime.Format(time.RFC3339)))
			} else {
				isValid = true
				WriteToFileDebug(fmt.Sprintf("Token is valid: %s", expireTime.Format(time.RFC3339)))
			}
		} else {
			WriteToFile("No expiration claim in token")
			isValid = false
		}
	} else {
		WriteToFile("Invalid token claims")
		isValid = false
	}
	return
}

// GetFormattedFloat - Transform the input (float) to a string with a given precision, and return "-" if value is zero */
func GetFormattedFloat(value float64, precision int) any {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%.*f", precision, value)
}
