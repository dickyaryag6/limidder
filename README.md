# limidder

Package to add rate limit in Golang

## Setup a config file in your project with `.yml` or `.json` format

`config.yml`
```
rate_limit:
  strategy: sliding_window
  config:
    "all":
      limit: 20
      duration: 30
    "GET /":
      limit: 10
      duration: 60
    "POST /":
      limit: 10
      duration: 10
```

## Convert the config values to this Golang struct
```
type Config struct {
	RateLimit RateLimit `yaml:"rate_limit"`
}

type RateLimit struct {
	Strategy string                      `yaml:"strategy"`
	Config   map[string]*limidder.Config `yaml:"config"`
}
```
## Intialize redis client before intializing rate limit
```
import (
  "github.com/go-redis/redis"
)

redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
	})
```
## Key Extraction Function
Create a function for extracting the key, the function need to receive `(key []string)` and return `(value string, err error)`<br>
This is example of extracting the key
```
func extractKey(key []string) (value string, err error) {
	return strconv.Itoa(rand.Intn(3)), nil
}
```

## Initialize Rate Limit
Here are the fields of the config struct you need to fulfill
1. `Extractor`<br>
    put this function `limidder.NewHTTPHeadersExtractor` which needs to receive the key extraction function above and also list of header variables if you need header values to get the key, e.g `limidder.NewHTTPHeadersExtractor(extractKey, "Authorization")`
2. `StrategyName`<br>
    put the rate limit algorithm of your choice. (at this point, the only option is `sliding_window`, more will be added soon)
3. `Config`<br>
    this is the field where we put the rate limit configurations, the things that you need to configure is the paths that needs to be rate-limited. 
    for example,
    ```
    routeRateLimitMap := make(map[string]*limidder.Config)
    routeRateLimitMap["GET /"] = &limidder.Config{
      Limit:    5,
      Duration: 10,
    }
    routeRateLimitMap["POST /"] = &limidder.Config{
      Limit:    5,
      Duration: 10,
    }
    ```
    if you want all paths to be rate-limited, put all as the key of the map
    ```
    routeRateLimitMap := make(map[string]*limidder.Config)
    routeRateLimitMap["all"] = &limidder.Config{
      Limit:    5,
      Duration: 10,
    }
    ```
4. `ApplyConfigToAllPaths`<br>
    set this to `true` if all paths need to be rate-limited, make sure the map in `Config` field contains "all" key
5. `ApplyUserRateLimitToAllPaths`<br>
    set this to `true` if the limit number of request has been reached, and needs to bed applied to all request to all paths

```
rateLimitConfig := limidder.RateLimiterConfig{
		Extractor:                    limidder.NewHTTPHeadersExtractor(extractKey, "Authorization"),
		StrategyName:                 cfg.RateLimit.Strategy,
		Config:                       cfg.RateLimit.Config,
		ApplyConfigToAllPaths:        false,
		ApplyUserRateLimitToAllPaths: false,
	}
```

pass rateLimitConfig and redis client to `InitRateLimiterMiddleware` to initialize rate limit
```
rateLimit := limidder.InitRateLimiterMiddleware(&rateLimitConfig, redisClient)
```

## Use rate limit as middleware<br>
if you use `github.com/go-chi/chi` package to create the routes like this, `router := chi.NewRouter()`, you can add the rate limit as middleware like this `router.Use(rateLimit.Handler)`


