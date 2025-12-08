# Arquitetura do Sistema UDP-Vote

Este documento apresenta uma visão clara e concisa da arquitetura do servidor de votação UDP desenvolvido em Go. Ele resume como o sistema funciona, seus componentes principais e o fluxo geral de comunicação.

---

## 🏗️ Visão Geral da Arquitetura

```
┌─────────────────────────────────────────────────────────────┐
│                           CLIENTES                           │
│   Client 1   Client 2   Client 3   ...   Client N            │
└─────────────────────────────────────────────────────────────┘
                              │ UDP
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     SERVIDOR (Port 9000)                     │
│                                                             │
│  Main Goroutine                                              │
│  └── listenAndServe()                                       │
│        └── go handleVote(packet)                            │
│                                                             │
│  Cada cliente → 1 goroutine própria                          │
│                                                             │
│  Estruturas protegidas por mutex:                            │
│    - clients: conexões ativas                                │
│    - votes: voto de cada cliente                             │
│    - voteCounts: contagem global                             │
│                                                             │
│  Broadcast:                                                  │
│    - Envio de status atual para todos os clientes           │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔄 Fluxo Básico do Sistema

### 1. Conexão do Cliente

* Cliente envia seu identificador ao servidor.
* Servidor registra o cliente e aguarda votos.

### 2. Votação

* Cliente envia: `VOTE X`
* Servidor:

  * Valida o voto.
  * Atualiza mapas protegidos.
  * Dispara broadcast com o estado atualizado.

### 3. Broadcast

* O servidor envia atualizações de votação para todos os clientes conectados.

---

## ⚙️ Concorrência e Estruturas Internas

### Goroutines principais

* **Main Goroutine** → Escuta pacotes UDP.
* **N Client Goroutines** → Uma goroutine por cliente para processar votos.

### Estrutura protegida por mutex

```
Server {
  mu          sync.Mutex
  clients     map[string]net.UDPAddr
  votes       map[string]string
  voteCounts  map[string]int
}
```

### Padrão de Acesso

* Todas as leituras/escritas nos mapas ocorrem dentro de `mu.Lock()` / `mu.Unlock()`.

---

## 📡 Broadcast e Problemas de Buffer

### Problemas de Buffer

* O uso de UDP pode levar a problemas de "Ghost Vote", onde votos podem ser enviados, mas não recebidos pelo servidor devido à perda de pacotes.
* O servidor deve lidar com a possibilidade de votos duplicados ou perdidos.

### Comparação com TCP

* Ao contrário do TCP, que garante a entrega de pacotes, o UDP não possui controle de fluxo, o que pode resultar em votos não contabilizados.
* O servidor deve implementar lógica para lidar com a inconsistência dos votos recebidos.

---

## 🚦 Ciclo de Vida do Cliente

```
DISCONNECTED → CONNECTED → REGISTERED → VOTED → DISCONNECTED
```

* Clientes recebem atualizações sempre que o estado global muda.
* Ao desconectar, o servidor remove o cliente do mapa.

---

## 🧱 Componentes do Sistema

### 1. Listener (Main Goroutine)

Escuta pacotes UDP e inicia goroutines para processar votos.

### 2. Processador de Voto

Realiza:

* Validação da opção.
* Atualização de `votes` e `voteCounts`.
* Disparo do broadcast.

---

## 🎯 Princípios Arquiteturais Utilizados

* **Goroutine-per-connection**: simples e altamente escalável.
* **Mutex apenas para memória**, nunca para operações de rede.
* **Mecanismos para lidar com perda de pacotes** e garantir a integridade dos votos.
* **I/O assíncrono** para máxima escalabilidade.

---

## 📊 Resumo de Performance

| Métrica                    | UDP                      |
| -------------------------- | ----------------------- |
| Garantia de entrega        | Não                     |
| Possibilidade de "Ghost Vote" | Alta                  |
| Escalabilidade             | Excelente               |