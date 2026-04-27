// Package eda provides the EDA worker for Austrian Marktkommunikation (MaKo).
// Re-exported types from internal/eda/types for convenient use.
package eda

import "github.com/lutzerb/eegabrechnung/internal/eda/types"

// Message is an alias for types.Message for backwards compatibility.
type Message = types.Message

// Direction and status constants.
const (
	DirectionInbound  = types.DirectionInbound
	DirectionOutbound = types.DirectionOutbound
	StatusPending     = types.StatusPending
	StatusSent        = types.StatusSent
	StatusAck         = types.StatusAck
	StatusError       = types.StatusError
	ProcessCRMsg      = types.ProcessCRMsg
)
