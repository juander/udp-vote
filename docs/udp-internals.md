# udp-internals.md

# UDP Internals

## 1. O que é UDP?

UDP (User Datagram Protocol) é um protocolo de comunicação sem conexão que permite a transmissão de datagramas entre dispositivos em uma rede. Ao contrário do TCP, o UDP não garante a entrega, a ordem ou a integridade dos pacotes, o que o torna mais leve e rápido, mas também mais suscetível a problemas como perda de pacotes.

## 2. Diferenças entre UDP e TCP

### Conexão

- **UDP**: Sem conexão. Não há necessidade de estabelecer uma conexão antes de enviar dados.
- **TCP**: Conexão orientada. Um handshake é necessário para estabelecer uma conexão antes da troca de dados.

### Garantia de entrega

- **UDP**: Não há garantias de entrega. Pacotes podem ser perdidos, duplicados ou entregues fora de ordem.
- **TCP**: Garante a entrega dos pacotes na ordem correta e sem duplicatas.

### Controle de fluxo

- **UDP**: Não possui controle de fluxo. O remetente pode enviar pacotes a qualquer momento, independentemente da capacidade do receptor.
- **TCP**: Implementa controle de fluxo para evitar sobrecarregar o receptor.

## 3. Implicações do uso de UDP para votação

Usar UDP para um sistema de votação apresenta desafios significativos:

- **Perda de pacotes**: Votos podem ser perdidos durante a transmissão, resultando em contagens de votos imprecisas.
- **Ghost Votes**: Um fenômeno onde um cliente pode enviar um voto, mas devido à perda de pacotes, o servidor pode não receber o voto, levando a uma situação em que o cliente acredita que votou, mas seu voto não é contabilizado.

## 4. Problemas potenciais relacionados à perda de pacotes

### 4.1. Votação não contabilizada

Se um cliente envia um voto e o pacote é perdido, o voto não será contabilizado, o que pode levar a resultados incorretos.

### 4.2. Duplicação de votos

Um cliente pode enviar o mesmo voto várias vezes se não receber uma confirmação do servidor, resultando em contagens de votos inflacionadas.

### 4.3. Ordem de entrega

Os pacotes podem chegar fora de ordem, o que pode complicar a lógica de processamento de votos, especialmente se a ordem for importante.

## 5. Estratégias para mitigar problemas de UDP

- **Confirmações de recebimento**: Implementar um mecanismo onde o servidor envia uma confirmação de recebimento para cada voto, permitindo que o cliente reenvie o voto se não receber a confirmação.
- **Timeouts e retransmissões**: Definir timeouts para reenvios de votos que não foram confirmados.
- **Registro de votos**: Manter um registro dos votos enviados e recebidos para evitar duplicações e garantir que todos os votos sejam contabilizados corretamente.

## 6. Conclusão

Embora o UDP ofereça vantagens em termos de velocidade e eficiência, seu uso em sistemas críticos como votação requer uma consideração cuidadosa das implicações de perda de pacotes e a implementação de estratégias para mitigar esses riscos. O fenômeno do "Ghost Vote" é um exemplo claro dos desafios que podem surgir ao usar UDP em um contexto onde a precisão e a confiabilidade são essenciais.