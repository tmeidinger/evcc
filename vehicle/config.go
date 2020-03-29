package vehicle

import (
	"strings"

	"github.com/andig/evcc/api"
)

// NewFromConfig creates charger from configuration
func NewFromConfig(log *api.Logger, typ string, other map[string]interface{}) api.Vehicle {
	var c api.Vehicle

	switch strings.ToLower(typ) {
	case "script", "exec":
		c = NewConfigurableFromConfig(log, other)
	case "audi":
		c = NewAudiFromConfig(log, other)
	case "tesla":
		c = NewTeslaFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid vehicle type '%s'", typ)
	}

	return c
}