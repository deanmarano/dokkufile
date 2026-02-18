package services

// Types lists all supported backing service types.
var Types = []string{
	"clickhouse",
	"couchdb",
	"elasticsearch",
	"mariadb",
	"meilisearch",
	"memcached",
	"mongo",
	"mysql",
	"nats",
	"postgres",
	"rabbitmq",
	"redis",
	"rethinkdb",
	"solr",
	"typesense",
}

// PluginURLs maps each service type to its dokku plugin install URL.
var PluginURLs = map[string]string{
	"clickhouse":    "https://github.com/dokku/dokku-clickhouse.git",
	"couchdb":       "https://github.com/dokku/dokku-couchdb.git",
	"elasticsearch": "https://github.com/dokku/dokku-elasticsearch.git",
	"mariadb":       "https://github.com/dokku/dokku-mariadb.git",
	"meilisearch":   "https://github.com/dokku/dokku-meilisearch.git",
	"memcached":     "https://github.com/dokku/dokku-memcached.git",
	"mongo":         "https://github.com/dokku/dokku-mongo.git",
	"mysql":         "https://github.com/dokku/dokku-mysql.git",
	"nats":          "https://github.com/dokku/dokku-nats.git",
	"postgres":      "https://github.com/dokku/dokku-postgres.git",
	"rabbitmq":      "https://github.com/dokku/dokku-rabbitmq.git",
	"redis":         "https://github.com/dokku/dokku-redis.git",
	"rethinkdb":     "https://github.com/dokku/dokku-rethinkdb.git",
	"solr":          "https://github.com/dokku/dokku-solr.git",
	"typesense":     "https://github.com/dokku/dokku-typesense.git",
}
