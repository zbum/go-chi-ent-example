---
title: 손으로 익히는 Go — chi와 ent로 REST API 만들기
site: https://www.manty.co.kr
status: draft
---

# 손으로 익히는 Go — chi와 ent로 REST API 만들기

이 글은 AI에게 프로젝트를 통째로 맡기지 않고, 내가 명령을 치고 파일을 고치며 Go HTTP 서버를 만든 기록이다. 라우터는 [chi](https://github.com/go-chi/chi), 데이터 계층은 [ent](https://entgo.io/docs/getting-started)를 쓴다. chi는 `net/http` 핸들러 그대로이고, ent는 스키마를 Go 코드로 정의한 뒤 타입 안전한 클라이언트를 생성한다. 둘 다 “마법”보다 읽히는 코드에 가깝다.

시작 시점의 저장소에는 GoLand 템플릿 `main.go`와 `go.mod`만 있다. 모듈 이름은 `go-chi-ent-example`, Go 버전은 `1.26`이다.

## 진행 상황

- [x] 1. 환경 확인
- [x] 2. Makefile 뼈대
- [x] 3. chi로 첫 HTTP 서버
- [x] 4. 미들웨어와 라우트 그룹
- [x] 5. JSON 핸들러
- [x] 6. httptest로 핸들러 테스트
- [x] 7. ent User 스키마
- [x] 8. 코드 생성과 SQLite 연결
- [ ] 9. User 생성·목록 API
- [ ] 10. URL 파라미터로 단건 조회

## 이 글의 약속

AI가 프로젝트를 통째로 만들지 않는다. 각 단계는 목표 / 왜 / 직접 치기 / 확인 으로 쓴다. 막히면 같은 단계를 더 잘게 나누고, 다음 단계로 건너뛰지 않는다.

## 단계

### 1. 환경 확인

**목표.** 지금 쓰는 Go와 모듈 선언이 이 실습의 바닥인지 눈으로 확인한다.

**왜.** 이후 `go get`, `go generate` 출력은 툴체인 버전에 묶인다. 시작 전에 `go.mod`를 읽지 않으면 다른 디렉터리에서 명령을 치고 있다고 착각하기 쉽다.

**직접 치기.**

```bash
go version
cat go.mod
ls
```

**확인.** `go version`에 `go1.26` 계열이 보이고, `go.mod` 첫 줄이 `module go-chi-ent-example` 이어야 한다. `main.go`가 루트에 있으면 이 프로젝트 루트에 있는 것이다.

**함정.** IDE 터미널이 다른 디렉터리일 수 있다. `pwd`가 저장소 루트가 아니면 이동한 뒤 다시 친다.

공식 문서: [Managing dependencies](https://go.dev/doc/modules/managing-dependencies)

**내가 본 출력.** `go version go1.26.5 darwin/arm64`. 모듈은 `go-chi-ent-example`, `go 1.26`. `ls main.go`로 루트의 템플릿 진입점을 확인했다. `go version`을 처음 `main`에서 쳤고, 이후 명령은 `feature/learn-chi-ent`에서 쳤다. 실습은 feature 브랜치에서 이어 간다.

### 2. Makefile 뼈대

**목표.** `test`와 세 플랫폼 빌드 타겟을 손으로 적는다.

**왜.** 이 컴퓨터의 Go 프로젝트는 루트 `Makefile`을 둔다. 지금은 바이너리보다 “어떻게 빌드할지 한곳에 적어 두는 습관”이 목적이다. 앱이 아직 Hello World여도 Makefile은 먼저 둔다.

**직접 치기.** 프로젝트 루트에 `Makefile`을 만들고 아래를 직접 입력한다.

```makefile
APP := go-chi-ent-example
BIN_DIR := dist

.PHONY: build-linux build-windows build-darwin build-all test clean

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(APP)-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(APP)-windows-amd64.exe .

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(APP)-darwin-arm64 .

build-all: build-linux build-windows build-darwin

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
```

그다음:

```bash
make test
```

**확인.** `make test`가 실패하지 않으면 된다. 테스트가 아직 없어도 `go test ./...`는 통과할 수 있다. `dist/`는 `.gitignore`에 이미 있다.

**함정.** 타겟 줄 아래 명령은 **탭**이어야 한다. 스페이스로 들여 쓰면 `missing separator`가 난다.

**내가 본 출력.** 첫 `make test`는 Makefile이 아니라 GoLand 템플릿 때문에 실패했다. `fmt.Println("Hello and welcome, %s!", s)` — `Println`은 서식을 해석하지 않는데 `%s`가 있어서 vet가 빌드를 막았다. `Printf`로 고친 뒤 `?    go-chi-ent-example  [no test files]` 로 통과했다. `build-darwin`은 `amd64`로 적어 두었고, 이 맥은 `arm64`다. 크로스 빌드는 나중에 맞춘다.

### 3. chi로 첫 HTTP 서버

**목표.** GoLand 템플릿 `main.go`를 지우고, chi 라우터로 `:3000`에서 `welcome`을 응답한다.

**왜.** chi는 `net/http`와 100% 호환된다. 핸들러 시그니처가 `func(w http.ResponseWriter, r *http.Request)` 그대로라서, 나중에 표준 라이브러리 미들웨어를 그대로 붙일 수 있다. 프레임워크 전용 컨텍스트를 먼저 배우지 않아도 된다.

**직접 치기.**

```bash
go get github.com/go-chi/chi/v5
```

`main.go`를 아래 내용으로 직접 고친다.

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	http.ListenAndServe(":3000", r)
}
```

터미널 하나에서 서버를 띄운다.

```bash
go run .
```

다른 터미널에서:

```bash
curl -i http://127.0.0.1:3000/
```

**확인.** 응답 본문이 `welcome`이고, 서버 터미널에 chi Logger 한 줄이 찍힌다. `go.mod`에 `github.com/go-chi/chi/v5`가 생긴다.

**함정.** import 경로는 반드시 `/v5`다. `github.com/go-chi/chi`만 쓰면 모듈 메이저 버전과 맞지 않는다. 포트가 이미 쓰이면 `bind: address already in use` — 이전 `go run`을 끄고 다시 한다.

공식 문서: [chi README](https://github.com/go-chi/chi)

**내가 본 출력.** `HTTP/1.1 200 OK`, `Content-Length: 7`, 본문 `welcome`. 줄 끝의 `%`는 응답에 개행이 없을 때 zsh가 넣는 `PROMPT_EOL_MARK`다. `go.mod`는 `github.com/go-chi/chi/v5 v5.3.2`.

### 4. 미들웨어와 라우트 그룹

**목표.** `RequestID`, `Recoverer`를 붙이고 `/health`와 `/v1/ping`을 나눈다.

**왜.** chi의 핵심은 라우터 트리를 쪼개는 것이다. `Use`는 그 라우터의 모든 요청에 미들웨어를 쌓고, `Route`는 접두어 아래 서브라우터를 만든다. 나중에 `/v1/users`를 붙일 자리가 여기서 생긴다.

**직접 치기.** `main.go`의 라우터 설정을 직접 고친다. 핵심만 보면 이렇다.

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
})

r.Route("/v1", func(r chi.Router) {
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
})
```

확인 요청:

```bash
curl -i http://127.0.0.1:3000/health
curl -i http://127.0.0.1:3000/v1/ping
```

**확인.** `/health` → `ok`, `/v1/ping` → `pong`. 로그에 request id가 보이면 `RequestID`가 붙은 것이다. `/ping`은 404여야 한다. 그룹이 `/v1` 아래만 받기 때문이다.

**함정.** `r.Route("/v1", ...)` 안의 `r`은 바깥 라우터와 다른 서브라우터다. 안쪽 `r.Get("/ping", ...)`의 실제 경로는 `/v1/ping`이다.

공식 문서: [chi Router interface](https://pkg.go.dev/github.com/go-chi/chi/v5#Router)

**내가 본 출력.** `/health` → `ok`, `/v1/ping` → `pong`, `/ping` → `404 page not found`. 404 본문에 개행이 있어 zsh `%`가 안 붙고, `X-Content-Type-Options: nosniff`는 Go `net/http` 기본 NotFound 응답이다. `/` welcome은 그대로 두었다.

### 5. JSON 핸들러

**목표.** `/v1/hello`가 `{"message":"hello"}`를 JSON으로 응답하게 한다. 핸들러를 `main` 밖으로 빼 함수로 둔다.

**왜.** REST API는 바이트 슬라이스보다 JSON을 주고받는다. `http.HandlerFunc`로 빼 두면 다음 단계에서 `httptest`로 서버 없이 검증할 수 있다. chi는 특별한 JSON 헬퍼를 강요하지 않는다. 표준 `encoding/json`이면 충분하다.

**직접 치기.**

```go
func helloJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "hello",
	})
}
```

라우트:

```go
r.Get("/hello", helloJSON) // /v1 그룹 안
```

```bash
curl -i http://127.0.0.1:3000/v1/hello
```

**확인.** `Content-Type`에 `application/json`이 있고 본문이 `{"message":"hello"}`다. 함수가 `main` 밖에 있어야 다음 단계 테스트가 같은 패키지에서 호출할 수 있다.

**함정.** `Encode`는 마지막에 개행을 넣는다. `{"message":"hello"}\n` 이 정상이다. `w.WriteHeader`를 `Encode` 이후에 호출하면 이미 200이 나간 뒤라 무시된다.

공식 문서: [encoding/json](https://pkg.go.dev/encoding/json)

**내가 본 출력.** `Content-Type: application/json; charset=UTF-8`, 본문 `{"message":"Hello, World!"}` (`Content-Length: 28` — JSON 27바이트 + `Encode`의 개행). 커리큘럼의 `"hello"` 대신 `"Hello, World!"`를 썼다. `helloJSON`은 `main` 밖 함수다.

### 6. httptest로 핸들러 테스트

**목표.** 서버를 띄우지 않고 `/v1/hello`를 테스트한다.

**왜.** `net/http/httptest`는 표준이다. chi 라우터는 `http.Handler`이므로 `httptest.NewRecorder`와 `httptest.NewRequest`만으로 충분하다. 공부용 프로젝트라도, 손으로 한 번 테스트를 쳐 두면 이후 CRUD에서 회귀를 바로 본다.

**직접 치기.** `main_test.go`를 새로 만들고 직접 입력한다. 라우터를 만드는 함수가 필요하면 `main.go`에서 `func newRouter() http.Handler`처럼 빼도 된다. 테스트는 그 함수를 호출한다.

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelloJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"message":"hello"}` {
		t.Fatalf("body = %q", body)
	}
}
```

```bash
make test
```

**확인.** `PASS`. 실패하면 먼저 실제 경로가 `/v1/hello`인지, `newRouter()`가 테스트와 같은 라우트를 쓰는지 본다.

**함정.** 테스트 파일이 `package main`이어야 같은 디렉터리의 `newRouter`를 부른다. `package main_test`로 두면 외부 테스트가 되어 소문자 함수를 못 본다.

공식 문서: [net/http/httptest](https://pkg.go.dev/net/http/httptest)

**내가 본 출력.** `ok      go-chi-ent-example      0.384s`. `newRouter()`를 `main.go`에서 빼 `ListenAndServe`에 넘겼고, `main_test.go`는 `package main`에서 같은 라우터로 `/v1/hello`를 검증했다.

### 7. ent User 스키마

**목표.** `ent` CLI로 `User` 스키마 파일을 만들고 `name`, `email` 필드를 직접 적는다.

**왜.** ent는 테이블을 마이그레이션 파일에서 시작하지 않는다. `ent/schema`에 Go struct로 모델을 적고, 생성기가 쿼리 빌더를 만든다. 스키마가 코드이므로 컴파일러가 필드 오타를 잡아 준다.

**직접 치기.**

```bash
go run -mod=mod entgo.io/ent/cmd/ent new User
```

생성된 `ent/schema/user.go`의 `Fields`를 직접 고친다.

```go
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("email").Unique(),
	}
}
```

`import`에 `"entgo.io/ent/schema/field"`를 넣는다.

**확인.** `ent/schema/user.go`가 있고, `Fields()`가 `nil`이 아니어야 한다. 이 단계에서는 아직 `go generate`를 돌리지 않는다. 스키마를 눈으로 읽은 뒤에 생성한다.

**함정.** `ent new`가 만드는 디렉터리는 `ent/schema/`다. 루트에 `user.go`를 만들지 말 것. `Unique()`는 이후 마이그레이션에서 UNIQUE 제약이 된다.

공식 문서: [ent getting started](https://entgo.io/docs/getting-started), [schema fields](https://entgo.io/docs/schema-fields)

**내가 본 출력.** `ent new User`가 `ent/schema/user.go`와 `ent/generate.go`를 만들었다. `Fields`는 `name`(`NotEmpty`)과 `email`(`Unique`). `Edges`는 아직 `nil`. `go.mod`에 `entgo.io/ent v0.14.6`.

### 8. 코드 생성과 SQLite 연결

**목표.** `go generate ./ent`로 클라이언트를 만들고, 프로세스 시작 시 SQLite에 연결해 스키마를 적용한다.

**왜.** 생성 코드(`ent/user.go`, `ent/client.go` 등)를 손으로 짜지 않는다. 우리가 소유하는 파일은 `ent/schema/*`와 `generate.go`다. 공부는 스키마를 읽고, 생성된 `Client`를 호출하는 쪽에 집중한다. 로컬 실습은 Postgres 설치 없이 SQLite면 충분하다.

**직접 치기.** `ent/generate.go`가 없으면 아래를 직접 만든다.

```go
package ent

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema
```

```bash
go generate ./ent
```

드라이버:

```bash
go get github.com/mattn/go-sqlite3
```

`main`에서 클라이언트를 연다. `_fk=1`은 SQLite에서 외래 키를 켜는 쿼리 파라미터다.

```go
client, err := ent.Open("sqlite3", "file:dev.db?_fk=1")
if err != nil {
    log.Fatalf("opening ent client: %v", err)
}
defer client.Close()

if err := client.Schema.Create(context.Background()); err != nil {
    log.Fatalf("creating schema: %v", err)
}
```

blank import가 필요하다.

```go
_ "github.com/mattn/go-sqlite3"
```

**확인.** `ent/client.go`가 생성된다. `go run .` 후 루트에 `dev.db`가 생긴다. `dev.db`는 git에 올리지 않는다 — `.gitignore`에 추가한다.

**함정.** `mattn/go-sqlite3`는 CGO가 필요하다. macOS에서 `CGO_ENABLED=0`이면 빌드가 실패한다. CGO를 쓰기 싫으면 같은 단계에서 `modernc.org/sqlite`로 바꿔도 된다. 그 경우 `ent.Open` 드라이버 이름과 blank import를 드라이버 문서에 맞춘다. 생성 파일은 보통 커밋한다. 스키마만 커밋하고 생성물을 빼면 다른 환경에서 `go generate`를 강제하게 된다.

공식 문서: [ent code generation](https://entgo.io/docs/code-gen), [ent migrations](https://entgo.io/docs/migrate)

**내가 본 출력.** `ent/client.go`, `ent/user.go` 등 생성물이 생겼고 `dev.db`(16384 bytes)가 루트에 있다. `main`에서 `ent.Open("sqlite3", "file:dev.db?_fk=1")` 후 `Schema.Create`를 호출한다. `newRouter()`에는 아직 client를 넘기지 않았다. `.gitignore`에 `dev.db`. CGO sqlite 드라이버 `mattn/go-sqlite3 v1.14.50`.

### 9. User 생성·목록 API

**목표.** `POST /v1/users`와 `GET /v1/users`를 ent 클라이언트에 연결한다.

**왜.** 스키마와 HTTP가 여기서 처음 만난다. 핸들러가 `*ent.Client`를 받도록 만들면, 테스트에서 sqlite memory를 주입할 수 있다. 전역 클라이언트를 숨기지 않는 편이 공부에 유리하다.

**직접 치기.** 요청 바디 예:

```json
{"name":"manty","email":"manty@example.com"}
```

생성은 이런 흐름이다.

```go
u, err := client.User.
    Create().
    SetName(body.Name).
    SetEmail(body.Email).
    Save(r.Context())
```

목록:

```go
users, err := client.User.Query().All(r.Context())
```

라우트는 `/v1` 그룹 안에 둔다.

```bash
curl -i -X POST http://127.0.0.1:3000/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"manty","email":"manty@example.com"}'

curl -i http://127.0.0.1:3000/v1/users
```

**확인.** POST가 201 또는 200과 함께 `id`를 돌려주고, GET 목록에 방금 만든 사용자가 있다. 같은 이메일로 두 번 POST하면 unique 제약 때문에 에러가 나야 한다. 그 에러를 500 그대로 내보낼지, 409로 바꿀지는 이 단계에서 결정해 본다.

**함정.** `r.Context()`를 쓰지 않고 `context.Background()`를 쓰면 클라이언트 취소가 쿼리까지 전달되지 않는다. JSON 디코드 실패는 400으로 돌려주는 편이 디버깅이 쉽다.

공식 문서: [ent CRUD](https://entgo.io/docs/crud)

### 10. URL 파라미터로 단건 조회

**목표.** `GET /v1/users/{id}`로 한 명을 조회한다. 없으면 404.

**왜.** chi의 경로 변수는 `chi.URLParam(r, "id")`로 읽는다. 문자열을 `strconv.Atoi`로 바꾼 뒤 `client.User.Get(ctx, id)`를 호출한다. REST에서 컬렉션과 단건을 나누는 최소 패턴이다.

**직접 치기.**

```go
r.Get("/users/{id}", getUser)
```

```go
id, err := strconv.Atoi(chi.URLParam(r, "id"))
// ...
u, err := client.User.Get(r.Context(), id)
```

`ent.IsNotFound(err)`이면 404.

```bash
curl -i http://127.0.0.1:3000/v1/users/1
curl -i http://127.0.0.1:3000/v1/users/9999
```

테스트도 하나 더 추가한다. 없는 id는 404인지 단언한다.

**확인.** 존재하는 id는 JSON 한 명, 없는 id는 404, 숫자가 아닌 id는 400. `make test`가 통과한다.

**함정.** 패턴은 `{id}`이고 읽는 키도 `"id"`다. `{userID}`로 적어 놓고 `"id"`를 읽으면 빈 문자열이 나온다. `Get`의 첫 인자는 요청 컨텍스트다.

공식 문서: [chi URLParam](https://pkg.go.dev/github.com/go-chi/chi/v5#URLParam), [ent IsNotFound](https://pkg.go.dev/entgo.io/ent#IsNotFound)

## 이 실습에서 다루지 않는 것

인증, Postgres, versioned migration(Atlas), 엔티티 관계(edge), 트랜잭션, Docker. 10단계를 손으로 끝낸 뒤에 고른다. 관계를 다음 주제로 삼는다면 ent getting started의 `Car` / `Group` edge부터 보면 된다.

## 참고

- [chi](https://github.com/go-chi/chi)
- [ent getting started](https://entgo.io/docs/getting-started)
- [Go modules](https://go.dev/doc/modules/managing-dependencies)
- [httptest](https://pkg.go.dev/net/http/httptest)
