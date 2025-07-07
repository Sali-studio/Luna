package handlers

import "time"

const (
	// Embed Colors
	ColorBlue   = 0x3498db
	ColorGreen  = 0x2ecc71
	ColorRed    = 0xf04747
	ColorOrange = 0xe67e22
	ColorGray   = 0x99aab5
	ColorTeal   = 0x58d68d

	// Config Keys
	ConfigKeyLog       = "log_config"
	ConfigKeyTempVC    = "temp_vc_config"

	// AI Server
	EnvPythonAIServerURL = "PYTHON_AI_SERVER_URL"

	// Audit Log Time Window
	AuditLogTimeWindow = 10 * time.Second
)
