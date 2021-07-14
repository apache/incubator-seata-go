package config_center

// ConfigurationListener
type ConfigurationListener interface {
	// Process the notification event once there's any change happens on the config
	Process(*ConfigChangeEvent)
}

// ConfigChangeEvent
type ConfigChangeEvent struct {
	Key   string
	Value interface{}
}
