## Godid

### 📋 Sobre

Godib é um auction house similar ao eBay. Neste projeto explorei o uso do WebSocket, 
sessões de chat, hashing de senha e segurança com CSRF tokens.


## 🛠️ Tecnologias
- API REST feita com Golanf e Chi
- SKRC;
- TERN;
- Docker
- R para live reloading
- PostgreSQL
- Websocket


## ⏳ Instalação e execução

Faça um clone desse repositório e acesse o diretório.

```bash
$ git clone https://github.com/JadnaSantos/app-theMovie.git
```

```bash
# Instalando as dependências
$ go mod tidy

# Executanto aplicação
$ air —build.cmd “go build -o ./bin/api ./cmd/api” — build.bin “./bin/api”

# Executando o Docker
docker compose -up
```

