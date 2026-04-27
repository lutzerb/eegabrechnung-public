// Package transport provides Transport interface implementations for EDA.
// The Transport interface itself is defined in internal/eda/types.
package transport

// This package re-exports nothing from types — callers should use
// types.Transport directly. The package exists to hold concrete
// implementations (MailTransport, PontonTransport).
