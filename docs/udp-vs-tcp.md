# UDP vs TCP: Diferenças Fundamentais

## Syscalls UDP

### Servidor
```c
socket(AF_INET, SOCK_DGRAM, 0)  // Cria socket UDP
bind(fd, &addr, sizeof(addr))    // Associa à porta
// Não precisa de listen()

recvfrom(fd, buffer, len, flags, &src_addr, &addr_len)  // Recebe pacote
sendto(fd, data, len, flags, &dest_addr, addr_len)      // Envia pacote
```

### Cliente
```c
socket(AF_INET, SOCK_DGRAM, 0)  // Cria socket
sendto(fd, data, len, ...)       // Envia (sem connect())
recvfrom(fd, buffer, len, ...)   // Recebe
```

## Diferenças Críticas

### 1. Confiabilidade

**TCP:**
- Garante entrega (ACK + retransmissão)
- Ordem garantida
- Conexão orientada

**UDP:**
- Sem garantia de entrega
- Sem ordem garantida
- Sem conexão (cada pacote independente)

### 2. Controle de Fluxo

**TCP:**
- Sliding window
- Backpressure: receptor controla velocidade
- write() BLOQUEIA se buffer cheio

**UDP:**
- Sem controle de fluxo
- sendto() NUNCA bloqueia
- SO descarta pacotes se buffer receptor cheio

## Fenômenos Demonstrados

### 1. Voto Fantasma

Cliente envia voto → pacote se perde → servidor nunca recebe → cliente não é notificado.

**Métrica:** `votos_enviados > votos_confirmados`

### 2. Buffer Overflow

Servidor envia broadcasts rápido → cliente lento não lê → buffer SO enche → SO descarta pacotes → gaps no SeqNum.

**Detecção:** Cliente vê SeqNum pular (ex: 5 → 12)

## Comparação

| Aspecto | TCP | UDP |
|---------|-----|-----|
| Entrega | Garantida | Best-effort |
| Ordem | Garantida | Não garantida |
| Controle fluxo | Sim (sliding window) | Não |
| Sistema trava | Sim (mutex+write bloqueante) | Não (sendto imediato) |
| Overhead | Alto | Baixo |
| Uso votação | Obrigatório | Inaceitável |

## Quando Usar

**TCP:**
- Votação
- Transações financeiras
- Transferência de arquivos
- Chat

**UDP:**
- Streaming de vídeo/áudio
- Jogos online
- VoIP
- DNS
