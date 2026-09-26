package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

type EVCCMode string

const (
	EVCCModeNormal EVCCMode = "normal"
	EVCCModeHold   EVCCMode = "hold"
	EVCCModeCharge EVCCMode = "charge"

	evccWatchdogTimeout = 90 * time.Second
)

var evccControl = struct {
	sync.RWMutex
	mode        EVCCMode
	lastCommand time.Time
}{
	mode: EVCCModeNormal,
}

type EVCCStatus struct {
	SOC              float64  `json:"soc"`
	BatteryPower     float64  `json:"batteryPower"`
	BatteryVoltage   float64  `json:"batteryVoltage"`
	BatteryCurrent   float64  `json:"batteryCurrent"`
	GridPower        float32  `json:"gridPower"`
	PVPower          float32  `json:"pvPower"`
	MinCellVoltage   float64  `json:"minCellVoltage"`
	MaxCellVoltage   float64  `json:"maxCellVoltage"`
	Mode             EVCCMode `json:"mode"`
	BMSValid         bool     `json:"bmsValid"`
	LastModeCommand  string   `json:"lastModeCommand,omitempty"`
}

func init() {
	addr := os.Getenv("EVCC_HTTP_ADDR")
	if addr == "" {
		addr = ":7070"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", handleEVCCStatus)
	mux.HandleFunc("/api/mode/normal", modeHandler(EVCCModeNormal))
	mux.HandleFunc("/api/mode/hold", modeHandler(EVCCModeHold))
	mux.HandleFunc("/api/mode/charge", modeHandler(EVCCModeCharge))

	go func() {
		log.Printf("evcc HTTP API listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("evcc HTTP API stopped: %v", err)
		}
	}()
}

func modeHandler(mode EVCCMode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		setEVCCMode(mode)
		w.WriteHeader(http.StatusNoContent)
	}
}

func setEVCCMode(mode EVCCMode) {
	evccControl.Lock()
	evccControl.mode = mode
	evccControl.lastCommand = time.Now()
	evccControl.Unlock()
	log.Printf("evcc battery mode: %s", mode)
}

func getEVCCMode() (EVCCMode, time.Time) {
	evccControl.RLock()
	mode := evccControl.mode
	lastCommand := evccControl.lastCommand
	evccControl.RUnlock()

	// Fail safe: if evcc stops refreshing a non-normal command, return to
	// the controller's original self-consumption behaviour.
	if mode != EVCCModeNormal && !lastCommand.IsZero() && time.Since(lastCommand) > evccWatchdogTimeout {
		setEVCCMode(EVCCModeNormal)
		return EVCCModeNormal, time.Now()
	}

	return mode, lastCommand
}

// applyEVCCMode modifies only the requested operating strategy. The existing
// BMS, SoC and cell-voltage safety limits in Compute() are applied afterwards.
func applyEVCCMode(requiredPower float32) float32 {
	mode, _ := getEVCCMode()

	switch mode {
	case EVCCModeHold:
		// Positive is charging in azur-modbus, negative is discharging.
		if requiredPower < 0 {
			return 0
		}
	case EVCCModeCharge:
		// Request maximum charge power. Existing controller limits reduce this
		// according to SoC and cell voltage.
		return 8000
	}

	return requiredPower
}

func handleEVCCStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode, lastCommand := getEVCCMode()

	pvPower := float32(0)
	for _, inverter := range inverters {
		pvPower += inverter.SolarPower
	}

	status := EVCCStatus{
		SOC:             batSoC,
		BatteryPower:    evccBatteryPower(),
		BatteryVoltage:  BmsData.BatteryVolts,
		BatteryCurrent:  BmsData.BatteryAmps,
		GridPower:       gridConsumption,
		PVPower:         pvPower,
		MinCellVoltage:  MinCellVolt,
		MaxCellVoltage:  MaxCellVolt,
		Mode:            mode,
		BMSValid:        BmsData.IsValid,
	}

	if !lastCommand.IsZero() {
		status.LastModeCommand = lastCommand.Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("evcc status encode error: %v", err)
	}
}

// evcc uses positive battery power for discharge and negative power for charge.
// The BMS state is used for the sign because the raw current representation is
// device-specific.
func evccBatteryPower() float64 {
	if !BmsData.IsValid {
		return 0
	}

	power := math.Abs(BmsData.BatteryVolts * BmsData.BatteryAmps)

	switch BmsData.State {
	case CHARGING:
		return -power
	case DISCHARGING:
		return power
	case IDLE:
		return 0
	default:
		log.Printf("unknown BMS state %v, reporting 0 W to evcc", BmsData.State)
		return 0
	}
}

func (m EVCCMode) GoString() string {
	return fmt.Sprintf("%q", string(m))
}
