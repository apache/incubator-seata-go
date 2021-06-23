package config_center

type ConfigurationListener interface {
	// Process the notification event once there's any change happens on the config
	Process(*ConfigChangeEvent)
}

type ConfigChangeEvent struct {
	Key   string
	Value interface{}
}
