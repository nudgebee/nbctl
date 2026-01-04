package config

import (
	"github.com/spf13/viper"
)

// WithViper sets keys temporarily on viper and returns a restore function.
func WithViper(overrides map[string]any) func() {
	// snapshot old values
	old := map[string]any{}
	for k := range overrides {
		old[k] = viper.Get(k)
	}

	for k, v := range overrides {
		viper.Set(k, v)
	}

	return func() {
		for k := range overrides {
			if old[k] == nil {
				viper.Set(k, "")
			} else {
				viper.Set(k, old[k])
			}
		}
	}
}
