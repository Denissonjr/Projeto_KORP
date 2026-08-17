# Projeto Korp — Desafio Docker, Golang, Redes e Ansible

Este repositório contém a solução completa do desafio: um serviço HTTP em
Golang (`http-server-projeto-korp`), a infraestrutura de containers
(NGINX, Prometheus, Grafana) e a automação de todo o provisionamento com
Ansible.

## Estrutura do repositório

```
.
├── http-server-projeto-korp/       # Código-fonte Go + Dockerfile do serviço
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── nginx/
│   └── http-server-projeto-korp.conf   # Proxy reverso 80 -> 8080
├── prometheus/
│   └── prometheus.yml                  # Configuração de scrape
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/datasource.yml  # Datasource Prometheus (auto)
│   │   └── dashboards/dashboard.yml    # Provider de dashboards (auto)
│   └── dashboards/
│       └── http-server-projeto-korp-dashboard.json  # Dashboard pronto
├── docker-compose.yml               # Orquestra app + nginx + prometheus + grafana
├── ansible/
│   ├── ansible.cfg
│   ├── inventory.ini
│   └── playbook.yml                 # Provisionamento ponta a ponta
└── README.md
```

## Parte 1 — Serviço e ambiente

### O serviço

- Escrito em Go puro (stdlib `net/http`), escuta na porta `8080`.
- `GET /projeto-korp` retorna:
  ```json
  { "nome": "Projeto Korp", "horario": "2026-08-17T13:00:00.123456789Z" }
  ```
  `horario` é resolvido a cada requisição com `time.Now().UTC()`.
- `GET /health`: endpoint de liveness/disponibilidade (usado pelo
  `HEALTHCHECK` do Docker).
- `GET /metrics`: métricas no formato Prometheus (ver Parte 2).

### Dockerfile

Build multi-stage: compila um binário estático em `golang:1.22-alpine` e
copia apenas o binário para uma imagem `alpine:3.19` mínima. `go mod tidy`
roda no build para resolver as dependências (requer acesso à internet no
momento do `docker build`, algo natural em CI/ambientes de build).

### Rede Docker e execução manual

```bash
# 1. Instalar Docker (veja a Parte 3 para fazer isso via Ansible)

# 2. Criar a rede bridge usada pelos containers
docker network create --driver bridge korp-network

# 3. Build + subir os containers
docker compose build
docker compose up -d

# 4. Testar
curl http://localhost:80/projeto-korp
# {"nome":"Projeto Korp","horario":"2026-08-17T13:00:00Z"}
```

O container `http-server-projeto-korp` **não expõe portas ao host** — só é
acessível dentro da rede `korp-network`. O container `nginx` mapeia
`80:80` e faz o proxy reverso para `http-server-projeto-korp:8080`,
usando o arquivo `nginx/http-server-projeto-korp.conf` montado em
`/etc/nginx/conf.d/`.

## Parte 2 — Monitoramento e observabilidade

Métricas expostas em `/metrics` (formato Prometheus), usando
`github.com/prometheus/client_golang`:

| Métrica                                                  | Tipo      | Finalidade                          |
|-----------------------------------------------------------|-----------|--------------------------------------|
| `http_server_projeto_korp_up`                              | gauge     | Disponibilidade do serviço (1 = up) |
| `http_server_projeto_korp_requests_total{method,path,status}` | counter | Volume de requisições                |
| `http_server_projeto_korp_request_duration_seconds`        | histogram | Latência das requisições             |

A disponibilidade também pode ser observada pela métrica nativa do
Prometheus `up{job="http-server-projeto-korp"}`, resultante do próprio
scrape (falha de scrape = serviço indisponível).

- **Prometheus** (`prometheus/prometheus.yml`) faz scrape de
  `http-server-projeto-korp:8080/metrics` a cada 15s.
- **Grafana** já vem com o datasource Prometheus e o dashboard
  provisionados automaticamente ao subir o container (arquivos em
  `grafana/provisioning/` e `grafana/dashboards/`) — nenhuma configuração
  manual é necessária. Acesse `http://localhost:3000` (usuário/senha
  `admin` / `admin`) e o dashboard **"Projeto Korp -
  http-server-projeto-korp"** já estará disponível, com painéis de:
  disponibilidade, volume/taxa de requisições, requisições por status
  code e latência p95.

## Parte 3 — Automação com Ansible

O playbook `ansible/playbook.yml` provisiona **tudo** com um único
comando: instala o Docker, cria a rede, copia os arquivos do projeto,
builda a imagem, sobe os containers via Docker Compose (app + NGINX +
Prometheus + Grafana, com o Grafana já provisionado automaticamente) e
valida o serviço com uma requisição HTTP real, imprimindo a resposta no
console.

```bash
cd ansible
ansible-playbook -i inventory.ini playbook.yml
```

Por padrão o inventário (`inventory.ini`) aponta para `localhost`
(`ansible_connection=local`), então o comando acima provisiona a própria
máquina onde o Ansible está rodando. Para provisionar um host remoto,
edite `inventory.ini` com o IP e usuário SSH desejados.

O playbook assume uma distribuição baseada em Debian/Ubuntu para a
instalação do Docker via `apt`; é o cenário mais comum para este tipo de
avaliação e pode ser adaptado para outras famílias (RHEL/Amazon Linux)
trocando o bloco de instalação por `dnf`/`yum`.

Ao final da execução, o playbook exibe no console a resposta JSON do
endpoint `/projeto-korp` e os endereços de acesso ao Prometheus e ao
Grafana.

## Decisões técnicas (resumo)

- **Go stdlib** em vez de framework web: o escopo é pequeno (um único
  endpoint + métricas) e a stdlib evita dependências desnecessárias.
- **Multi-stage Dockerfile**: imagem final mínima (Alpine + binário
  estático), reduzindo superfície de ataque e tamanho.
- **Rede externa (`external: true`) no compose**: a rede é criada
  explicitamente (manualmente ou pelo Ansible) antes do `docker compose
  up`, atendendo ao requisito de criação de rede como etapa própria, em
  vez de deixar o Compose criá-la implicitamente.
- **Sem publicação de porta no serviço Go**: apenas o NGINX conversa com
  o host; o serviço de aplicação só é alcançável dentro da rede Docker.
- **Provisioning automático do Grafana**: datasource e dashboard sobem
  prontos junto com o container, sem passos manuais (atendendo ao bônus
  do desafio).
- **Ansible sem collections externas**: usa apenas módulos
  `ansible.builtin`, simplificando a execução com um único comando sem
  necessidade de `ansible-galaxy collection install` prévio.
