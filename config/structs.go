package config

type Config struct {
	SentryIoDsn        string `json:"sentry_io_dsn"`
	AwsRegion          string `json:"aws_region"`
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
	BucketName         string `json:"bucket_name"`
	ListenPort         int    `json:"listen_port"`
}
