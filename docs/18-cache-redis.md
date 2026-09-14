# 18 — Redis: cache va refresh token store

```bash
go get github.com/redis/go-redis/v9
```

## Client — `pkg/redis/redis.go`

```go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

func NewClient(ctx context.Context, addr string) (*redis.Client, error) {
    c := redis.NewClient(&redis.Options{
        Addr:         addr,
        DialTimeout:  3 * time.Second,
        ReadTimeout:  2 * time.Second,
        WriteTimeout: 2 * time.Second,
    })
    pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    if err := c.Ping(pingCtx).Err(); err != nil {
        return nil, fmt.Errorf("redis: ping: %w", err)
    }
    return c, nil
}
```

## Refresh token store — `internal/repository/redis/refresh_token_store.go`

```go
package redis

import (
    "context"
    "errors"
    "fmt"
    "strconv"
    "time"

    "github.com/redis/go-redis/v9"

    "github.com/yodzafar/myservice/internal/domain"
    "github.com/yodzafar/myservice/internal/service"
)

var _ service.RefreshTokenStore = (*RefreshTokenStore)(nil)

type RefreshTokenStore struct {
    rdb *redis.Client
}

func NewRefreshTokenStore(rdb *redis.Client) *RefreshTokenStore {
    return &RefreshTokenStore{rdb: rdb}
}

func key(token string) string { return "refresh:" + token }

func (s *RefreshTokenStore) Save(ctx context.Context, token string, userID int64, ttl time.Duration) error {
    if err := s.rdb.Set(ctx, key(token), userID, ttl).Err(); err != nil {
        return fmt.Errorf("refresh store: save: %w", err)
    }
    return nil
}

func (s *RefreshTokenStore) Get(ctx context.Context, token string) (int64, error) {
    val, err := s.rdb.Get(ctx, key(token)).Result()
    if errors.Is(err, redis.Nil) {
        return 0, domain.ErrInvalidToken
    }
    if err != nil {
        return 0, fmt.Errorf("refresh store: get: %w", err)
    }
    return strconv.ParseInt(val, 10, 64)
}

func (s *RefreshTokenStore) Delete(ctx context.Context, token string) error {
    return s.rdb.Del(ctx, key(token)).Err()
}
```

Bir user'ning barcha sessiyalarini o'chirish kerak bo'lsa: qo'shimcha `SADD user_tokens:<userID> <token>` set saqlang.

## Cache-aside pattern (o'qishni tezlashtirish)

`service/ports.go`:

```go
type Cache interface {
    Get(ctx context.Context, key string, dst any) error // topilmasa ErrCacheMiss
    Set(ctx context.Context, key string, val any, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
}
```

`internal/repository/redis/cache.go`:

```go
var ErrCacheMiss = errors.New("cache miss")

type Cache struct{ rdb *redis.Client }

func (c *Cache) Get(ctx context.Context, key string, dst any) error {
    b, err := c.rdb.Get(ctx, key).Bytes()
    if errors.Is(err, redis.Nil) {
        return ErrCacheMiss
    }
    if err != nil {
        return err
    }
    return json.Unmarshal(b, dst)
}

func (c *Cache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
    b, err := json.Marshal(val)
    if err != nil {
        return err
    }
    return c.rdb.Set(ctx, key, b, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
    return c.rdb.Del(ctx, keys...).Err()
}
```

Service'da:

```go
func (s *ProductService) Get(ctx context.Context, id int64) (*domain.Product, error) {
    key := fmt.Sprintf("product:%d", id)

    var cached domain.Product
    if err := s.cache.Get(ctx, key, &cached); err == nil {
        return &cached, nil
    } else if !errors.Is(err, redis.ErrCacheMiss) {
        s.log.WarnContext(ctx, "cache get failed", "err", err) // cache xatosi → DB'ga o'tamiz, 500 emas
    }

    p, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if err := s.cache.Set(ctx, key, p, 5*time.Minute); err != nil {
        s.log.WarnContext(ctx, "cache set failed", "err", err)
    }
    return p, nil
}

func (s *ProductService) Update(ctx context.Context, ...) (*domain.Product, error) {
    // ... update
    _ = s.cache.Delete(ctx, fmt.Sprintf("product:%d", id)) // invalidation
    return p, nil
}
```

## Qoidalar

1. Cache **ixtiyoriy** — Redis o'chsa servis DB bilan ishlayveradi (faqat log).
2. Kalit nomlash: `<entity>:<id>`, `<entity>:list:<hash(filter)>`. Prefiks bilan — `KEYS product:*` (faqat debug'da; prod'da `SCAN`).
3. Har doim TTL. TTL'siz kalit — memory leak.
4. Yozishda invalidatsiya (`Delete`), yangilangan qiymatni yozish emas (race).
5. Domain struct'ni JSON qilish — domain'da `json` teg yo'q, lekin `encoding/json` public field'larni default nom bilan yozadi. Muammo bo'lmaydi; aniqroq bo'lsin desangiz cache'ga DTO yozing.
6. Rate limit, session, distributed lock (`SET NX`), pub/sub — Redis'ning boshqa vazifalari.
