# UDP Vote - Guia de Execução

## Executar o Servidor

```bash
go run cmd/server/main.go
```

## Executar o Cliente

O cliente requer um nome como argumento:

```bash
# Linux/Mac
go run cmd/client/main.go Alice

# Windows
go run .\cmd\client\main.go Alice
```

### Comandos do Cliente

Após conectar, você pode usar:

- `VOTE A` - Votar na opção A
- `VOTE B` - Votar na opção B
- `VOTE C` - Votar na opção C
- `STATS` - Ver estatísticas (votos fantasma, packets perdidos)
- `QUIT` - Sair e exibir estatísticas finais

## Executar Teste de Carga

```bash
# Linux/Mac
go run test/loadtest.go

# Windows
go run .\test\loadtest.go
```

O teste demonstra:
- **Votos Fantasma**: Votos enviados mas não confirmados
- **Buffer Overflow**: Packets perdidos por clientes lentos

## Exemplo de Uso Completo

### Terminal 1 - Servidor
```bash
go run cmd/server/main.go
```

### Terminal 2 - Cliente 1
```bash
go run cmd/client/main.go Alice
>> VOTE A
>> STATS
```

### Terminal 3 - Cliente 2
```bash
go run cmd/client/main.go Bob
>> VOTE B
```

### Terminal 4 - Teste de Carga
```bash
go run test/loadtest.go
```

## Estrutura do Projeto

```
cmd/
  client/main.go    - Cliente UDP
  server/main.go    - Servidor UDP
internal/
  server/
    server.go       - Lógica do servidor
    types.go        - Tipos compartilhados
test/
  loadtest.go       - Teste de carga UDP
```

## Syscalls UDP Utilizados

**Servidor:**
- `socket(AF_INET, SOCK_DGRAM, 0)` - Cria socket UDP
- `bind()` - Associa à porta 9000
- `recvfrom()` - Recebe pacotes
- `sendto()` - Envia pacotes (não bloqueia)

**Cliente:**
- `socket(AF_INET, SOCK_DGRAM, 0)` - Cria socket UDP
- `sendto()` - Envia votos
- `recvfrom()` - Recebe confirmações e broadcasts
