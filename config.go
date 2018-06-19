package main

type Config struct {
	TelegramToken string `yaml:"telegram_token"`
	AlertManagerAddress string `yaml:"alertmanager_address"`
	TemplatePath  string `yaml:"template_path"`
	TimeZone      string `yaml:"time_zone"`
	TimeOutFormat string `yaml:"time_outdata"`
	SplitChart    string `yaml:"split_token"`
	AuthorizedChatIds []int64 `yaml:"authorized_chat_ids"`
}
