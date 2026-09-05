---
title: 손으로 익히는 ent — edge와 트랜잭션
site: https://www.manty.co.kr
status: ready
---

# 손으로 익히는 ent — edge와 트랜잭션

이 글은 chi와 ent로 User CRUD를 만든 다음 편이다. 1편에서는 필드만 있는 `User`를 HTTP에 붙였다. 이번엔 엔티티 사이 관계(edge)와, 두 쓰기를 한 단위로 묶는 트랜잭션을 손으로 친다. 공식 문서의 `Car` / `Group` 예제를 이 저장소의 `name`·`email` User에 맞춰 옮긴다.

시작 시점의 저장소에는 `POST/GET /v1/users`, `GET /v1/users/{id}`가 있다. `ent/schema/user.go`의 `Edges()`는 `nil`이다. SQLite는 `file:dev.db?_fk=1`이다.

## 진행 상황

- [x] 1. 스키마와 라우트 확인
- [x] 2. Car 스키마
- [x] 3. User에서 cars로 O2M
- [x] 4. Car에서 owner로 역방향 edge
- [x] 5. 사용자에게 차를 붙이는 API
- [x] 6. Group과 M2M
- [x] 7. 트랜잭션으로 사용자와 차를 같이 만들기
- [x] 8. 롤백을 테스트로 확인

## 이 글의 약속

AI가 프로젝트를 통째로 만들지 않는다. 각 단계는 목표 / 왜 / 직접 치기 / 확인 으로 쓴다. 막히면 같은 단계를 더 잘게 나누고, 다음 단계로 건너뛰지 않는다.

## 단계

### 1. 스키마와 라우트 확인

**목표.** 1편이 남긴 User와, edge가 아직 없는 상태를 눈으로 확인한다.

**왜.** 이후 `ent new`와 `go generate`는 지금 스키마 위에 쌓인다. `Edges()`가 이미 채워져 있다고 가정하면 생성 코드와 핸들러가 어긋난다.

**직접 치기.**

```bash
cat ent/schema/user.go
ls ent/schema
```

**확인.** `Fields`에 `name`, `email`이 있고 `Edges()`가 `nil`을 반환한다. `ent/schema`에는 `user.go`만 있다.

**함정.** `ent/user.go`는 생성기다. 관계를 넣는 파일은 `ent/schema/user.go`다.

공식 문서: [ent getting started](https://entgo.io/docs/getting-started)

**내가 본 출력.** `ent/schema/user.go`의 `Fields`는 `name`(`NotEmpty`), `email`(`Unique`). `Edges()`는 `nil`. `ls ent/schema`는 `user.go`만.

### 2. Car 스키마

**목표.** `Car` 엔티티를 CLI로 만들고 `model`, `registered_at` 필드를 직접 적는다.

**왜.** edge는 두 스키마가 있어야 선언할 수 있다. 공식 문서도 User 다음에 Car를 만든다. 필드를 먼저 고정한 뒤 관계를 붙인다.

**직접 치기.**

```bash
go run -mod=mod entgo.io/ent/cmd/ent new Car
```

`ent/schema/car.go`의 `Fields`를 직접 고친다.

```go
func (Car) Fields() []ent.Field {
	return []ent.Field{
		field.String("model").NotEmpty(),
		field.Time("registered_at"),
	}
}
```

`import`에 `"entgo.io/ent/schema/field"`를 넣는다. 이 단계에서는 아직 `Edges`와 `go generate`를 하지 않는다.

**확인.** `ent/schema/car.go`가 있고 `Fields()`가 `nil`이 아니다.

**함정.** `ent new Car`는 `ent/schema/car.go`를 만든다. 루트에 `car.go`를 만들지 말 것. `Time`은 `time.Time`이다. JSON으로 넣을 때는 RFC3339 문자열을 나중에 디코드한다.

공식 문서: [getting started — Car](https://entgo.io/docs/getting-started#add-your-first-edge-relation), [schema fields](https://entgo.io/docs/schema-fields)

**내가 본 출력.** `ent new Car`가 `ent/schema/car.go`를 만들었다. `Fields`는 `model`(`NotEmpty`)과 `registered_at`(`Time`). `Edges`는 아직 `nil`. `ent/schema`는 `car.go`, `user.go`.

### 3. User에서 cars로 O2M

**목표.** User가 차를 여러 대 갖도록 `edge.To("cars", Car.Type)`를 넣고 코드를 생성한다.

**왜.** `edge.To`를 선언한 스키마가 관계를 소유한다. Unique를 붙이지 않으면 O2M이다. 생성기가 `AddCars`, `QueryCars`를 만든다.

**직접 치기.** `ent/schema/user.go`의 `Edges`를 고친다.

```go
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cars", Car.Type),
	}
}
```

`import`에 `"entgo.io/ent/schema/edge"`를 넣는다.

```bash
go generate ./ent
```

서버를 한 번 띄워 스키마를 적용한다.

```bash
go run .
```

**확인.** 생성 코드에 `QueryCars` / `AddCars`가 생긴다. `go run .`이 실패하지 않으면 `Schema.Create`가 `cars` 테이블을 추가한 것이다.

**함정.** 스키마만 바꾸고 `go generate`를 빼먹으면 `Car.Type`은 컴파일돼도 클라이언트가 옛 코드다. 기존 `dev.db`가 있어도 `_fk=1`이면 외래 키 제약이 산다. 생성이 이상하면 `dev.db`를 지우고 다시 `go run .`한다.

공식 문서: [O2M two types](https://entgo.io/docs/schema-edges#o2m-two-types)

**내가 본 출력.** User `Edges`에 `edge.To("cars", Car.Type)`. `go generate ./ent` 후 `ent/user.go`에 `QueryCars`, `ent/user_create.go`에 `AddCars`. `ent/car.go` 등 생성물이 생겼다. `go build ./...` 통과.

### 4. Car에서 owner로 역방향 edge

**목표.** Car에서 차 주인 User를 `owner`로 읽게 한다. DB에 새 간선을 만들지 않는다.

**왜.** `edge.From` + `Ref("cars")`는 User 쪽 `cars`의 뒷면이다. `Unique()`를 붙이면 차 한 대에 주인 한 명이다. 그래프에서 양쪽으로 걷는다.

**직접 치기.** `ent/schema/car.go`의 `Edges`를 고친다.

```go
func (Car) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("cars").
			Unique(),
	}
}
```

`import`에 `"entgo.io/ent/schema/edge"`를 넣는다.

```bash
go generate ./ent
```

**확인.** `Car`에 `QueryOwner` / `SetOwner`가 생긴다. `Ref("cars")`의 문자열이 User 스키마의 edge 이름과 같아야 한다.

**함정.** `Ref("car")`처럼 단수로 적으면 생성기가 실패한다. `Unique()`를 빼면 차 한 대가 여러 주인을 갖는 M2M이 된다. O2M의 역방향은 Unique가 맞다.

공식 문서: [inverse edge](https://entgo.io/docs/getting-started#add-your-first-inverse-edge-backref)

**내가 본 출력.** Car `Edges`는 `edge.From("owner", User.Type).Ref("cars").Unique()`. `go generate` 후 `QueryOwner`, `SetOwner`, `SetOwnerID`. `go build ./...` 통과.

### 5. 사용자에게 차를 붙이는 API

**목표.** `POST /v1/users/{id}/cars`로 차를 만들고 주인에 연결한다. `GET /v1/users/{id}/cars`로 그 사용자 차를 목록한다.

**왜.** 스키마와 HTTP가 여기서 다시 만난다. `SetOwnerID` 또는 `SetOwner`로 FK를 채운다. 목록은 `client.User.QueryCars` 또는 `client.Car.Query().Where(car.HasOwnerWith(...))`다.

**직접 치기.** 요청 바디 예:

```json
{"model":"Tesla","registered_at":"2026-08-30T12:00:00+09:00"}
```

생성 흐름:

```go
id, err := strconv.Atoi(chi.URLParam(r, "id"))
c, err := client.Car.
	Create().
	SetModel(body.Model).
	SetRegisteredAt(body.RegisteredAt).
	SetOwnerID(id).
	Save(r.Context())
```

목록:

```go
cars, err := client.User.QueryCars(user).All(r.Context())
```

또는 User를 먼저 가져온 뒤 `u.QueryCars().All(ctx)`를 쓴다. 사용자가 없으면 404.

라우트는 `/v1` 안에 둔다.

```go
r.Post("/users/{id}/cars", createCar(client))
r.Get("/users/{id}/cars", listCars(client))
```

```bash
curl -i -X POST http://127.0.0.1:3000/v1/users/1/cars \
  -H 'Content-Type: application/json' \
  -d '{"model":"Tesla","registered_at":"2026-08-30T12:00:00+09:00"}'

curl -i http://127.0.0.1:3000/v1/users/1/cars
```

**확인.** POST가 201 또는 200과 `id`·`model`을 준다. GET 목록에 그 차가 있다. 없는 user id로 POST하면 404 또는 FK 에러를 이 단계에서 고른다. 없는 user의 GET은 404.

**함정.** `registered_at` JSON은 `time.Time`으로 디코드한다. 키를 `registeredAt`으로 적으면 제로 타임이 들어간다. `SetOwnerID`는 generate 이후의 필드 이름이다. 생성 코드의 setter가 `SetOwnerID`가 아니면 `SetOwner(u)`로 User를 넘긴다.

공식 문서: [CRUD](https://entgo.io/docs/crud), [chi URLParam](https://pkg.go.dev/github.com/go-chi/chi/v5#URLParam)

**내가 본 출력.** `POST /v1/users/{id}/cars`, `GET /v1/users/{id}/cars`. 요청 타입은 `CarCreateRequest`(`json:"model"`, `json:"registered_at"`). 처음 태그는 `registeredAt`이라 POST 응답이 `"registered_at":"0001-01-01T00:00:00Z"`였다. 태그를 `registered_at`으로 고친 뒤 두 번째 차는 `2026-08-30T12:00:00+09:00`.

생성은 `SetOwnerID(id)`. 목록은 `User.Get` 후 `u.QueryCars().All`. 없는 user는 `404` `ent: user not found`.

목록 핸들러를 생성과 같이 `201`로 뒀다가 `StatusOK`로 바꿨다. `go run`을 다시 켜야 GET이 200으로 보인다.

```
GET /v1/users/1/cars → 차 두 대 (id 1 제로 타임 Tesla, id 2 시각이 있는 Tesla)
GET /v1/users/9999/cars → 404
POST → 201 {"id":1,"model":"Tesla",...}
```

### 6. Group과 M2M

**목표.** `Group`을 만들고 Group `users` ↔ User `groups` 다대다를 선언한다.

**왜.** O2M만으로는 조인 테이블이 안 보인다. M2M은 `edge.To`를 가진 쪽이 소유하고, 반대는 `edge.From` + `Ref`다. Unique를 붙이지 않는다.

**직접 치기.**

```bash
go run -mod=mod entgo.io/ent/cmd/ent new Group
```

`ent/schema/group.go`:

```go
func (Group) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
	}
}

func (Group) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type),
	}
}
```

User `Edges`에 역방향을 더한다. 기존 `cars`는 유지한다.

```go
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cars", Car.Type),
		edge.From("groups", Group.Type).
			Ref("users"),
	}
}
```

```bash
go generate ./ent
```

테스트나 `main`에서 그룹 하나를 만들고 사용자를 붙인다. HTTP를 붙이든 테스트만 하든 이 단계에서 고른다. 최소 확인은 아래면 된다.

```go
g, err := client.Group.Create().SetName("GitHub").AddUsers(u).Save(ctx)
users, err := g.QueryUsers().All(ctx)
```

**확인.** `go generate`가 통과한다. 그룹에 넣은 사용자가 `QueryUsers`에 나온다. User에서 `QueryGroups`로 같은 그룹이 보인다.

**함정.** User에 `Ref("users")`를 빼거나 이름을 `user`로 적으면 생성 실패다. Group 쪽 `To("users")`와 같아야 한다. `Unique()`를 양쪽에 붙이면 O2O가 된다.

공식 문서: [M2M two types](https://entgo.io/docs/schema-edges#m2m-two-types)

**내가 본 출력.** Group `edge.To("users", User.Type)`, User는 `edge.From("Groups", Group.Type).Ref("users")`. 커리큘럼의 `groups` 대신 `Groups`. `go generate` 후 `TestGroup`에서 memory sqlite로 User를 만들고 `AddUsers`로 GitHub 그룹에 붙였다. `g.QueryUsers()`, `u.QueryGroups()`까지 호출했고 `go test -v`에서 `TestGroup` PASS. HTTP 핸들러는 만들지 않았다. 테스트는 `err`와 `nil`만 본다. `All()`이 빈 슬라이스를 주면 그것도 non-nil이라 단언을 통과한다.

### 7. 트랜잭션으로 사용자와 차를 같이 만들기

**목표.** 사용자 생성과 첫 차 생성을 한 트랜잭션으로 묶는다. 차 만들기가 실패하면 사용자도 남지 않게 한다.

**왜.** `client.User.Create`와 `client.Car.Create`를 따로 `Save`하면 중간 실패 때 사용자만 남는다. `client.Tx(ctx)`로 얻은 `*ent.Tx`의 `User`/`Car`를 쓴 뒤 `Commit` 또는 `Rollback`한다.

**직접 치기.** `POST /v1/users-with-car`처럼 새 경로를 두거나, 기존 생성 API를 바꿔도 된다. 바디 예:

```json
{"name":"manty","email":"manty2@example.com","model":"Tesla"}
```

핵심:

```go
tx, err := client.Tx(r.Context())
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
u, err := tx.User.Create().
	SetName(body.Name).
	SetEmail(body.Email).
	Save(r.Context())
if err != nil {
	_ = tx.Rollback()
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
_, err = tx.Car.Create().
	SetModel(body.Model).
	SetRegisteredAt(time.Now()).
	SetOwner(u).
	Save(r.Context())
if err != nil {
	_ = tx.Rollback()
	http.Error(w, err.Error(), http.StatusBadRequest)
	return
}
if err := tx.Commit(); err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
```

`model`이 빈 문자열이면 `NotEmpty` 검증이 실패한다. 그때 사용자를 GET해도 없어야 한다.

공식 문서의 `rollback` 헬퍼를 그대로 옮겨도 된다.

```bash
curl -i -X POST http://127.0.0.1:3000/v1/users-with-car \
  -H 'Content-Type: application/json' \
  -d '{"name":"tx-ok","email":"tx-ok@example.com","model":"Tesla"}'

curl -i -X POST http://127.0.0.1:3000/v1/users-with-car \
  -H 'Content-Type: application/json' \
  -d '{"name":"tx-fail","email":"tx-fail@example.com","model":""}'

curl -i http://127.0.0.1:3000/v1/users
```

**확인.** 성공 POST는 사용자와 차가 둘 다 있다. `model`이 빈 요청은 4xx/5xx이고, `tx-fail@example.com` 사용자는 목록에 없다.

**함정.** 실패 뒤 `Commit`을 호출하지 말고 `Rollback`한다. 트랜잭션 안에서 만든 엔티티로 edge를 다시 쿼리하려면 커밋 후 `Unwrap()`이 필요하다. 커밋 전에 `Unwrap`하면 패닉이다. 바깥 `client`로 쓰지 말고 `tx.User` / `tx.Car`를 쓴다.

공식 문서: [transactions](https://entgo.io/docs/transactions)

**내가 본 출력.** `POST /v1/users-with-car`. `client.Tx` / `tx.User.Create` / `tx.Car.Create` / `tx.Commit`. `Tx`와 `Commit` 실패는 500. 차 `Save` 실패는 `Rollback` 후 500.

성공: `201` `{"id":6,"name":"tx-ok2","email":"tx-ok-1788620870@example.com","edges":{}}`. 응답에는 차가 없다. `GET /v1/users/6/cars`에 Tesla. 빈 model: `500` `validator failed for field "Car.model"`. `tx-fail` 이메일은 목록에 없다. `WithTx` 헬퍼는 쓰지 않았고, 분기마다 `Rollback()`을 직접 친다.

### 8. 롤백을 테스트로 확인

**목표.** 서버를 띄우지 않고, 빈 `model`이면 사용자 행이 안 남는 것을 테스트한다.

**왜.** curl만으로는 다음 수정이 롤백을 깨도 늦게 안다. sqlite memory 클라이언트에 스키마를 적용하고 같은 핸들러를 호출한다.

**직접 치기.** `main_test.go`에 테스트를 추가한다. 클라이언트를 연다.

```go
client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
```

`Schema.Create` 후 `newRouter(client)`로 POST 두 번을 보낸다. 하나는 유효한 model, 하나는 빈 model. 빈 model 응답이 2xx가 아니고, `GET /v1/users`에 그 이메일이 없다.

```bash
make test
```

**확인.** `make test` PASS. 실패 케이스에서 `rec.Code`가 4xx 또는 5xx다. 성공 케이스는 사용자가 생긴다.

**함정.** `newRouter(nil)`로 이 테스트를 치면 `Tx`에서 패닉이다. memory DSN에 `_fk=1`이 없으면 롤백과 FK 동작이 파일 DB와 달라질 수 있다. `t.Context()`를 `Schema.Create`에 넘겨도 된다.

공식 문서: [httptest](https://pkg.go.dev/net/http/httptest), [transactions](https://entgo.io/docs/transactions)

**내가 본 출력.** `TestRollback`. 처음엔 `GET /v1/users/99`와 `newRouter(nil)`이라 패닉을 Recoverer가 500으로 가렸다. POST로 바꾼 뒤에는 GET 405로도 PASS가 났다. 빈 model POST + `newRouter(nil)`은 `client.Tx` nil 패닉 → 500이라 또 PASS.

`newRouter(client)`로 바꾼 뒤:

```
POST /v1/users-with-car → 500 84B
GET /v1/users → 200 3B  (`[]` + 개행)
```

본문에 `tx-fail@example.com` 없음. `make test` PASS. 성공(커밋) POST는 이 테스트에 넣지 않았다.

## 이 실습에서 다루지 않는 것

eager load(`WithCars`), edge field, edge schema, 트랜잭션 훅, isolation level, Atlas versioned migration, Postgres. 8단계를 손으로 끝낸 뒤에 고른다. 다음 편을 마이그레이션으로 잡을 거면 [versioned migrations](https://entgo.io/docs/versioned-migrations)부터 보면 된다.

## 참고

- 이 글의 저장소: [github.com/zbum/go-chi-ent-example](https://github.com/zbum/go-chi-ent-example)
- 1편: [손으로 익히는 Go — chi와 ent로 REST API 만들기](./LEARN.md)
- [ent getting started](https://entgo.io/docs/getting-started)
- [schema edges](https://entgo.io/docs/schema-edges)
- [transactions](https://entgo.io/docs/transactions)
