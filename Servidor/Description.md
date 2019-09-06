##### Este arquivo tem como função definir metas e marcar metas já realizadas

#### Inicialização do servidor

> "go run server.go 'nome'" 
* onde o servidor busca no arquivo a porta referente ao nome;    

#### Descrição do funcionamento do servidor
* Servidor lê do arquivo "serverconfig.json"
* Seleciona a porta para escutar novos clientes
* Servidor inicia um Lobby onde serão criadas novas salas .
* Seleciona a porta para escutar novos servidores (também é definida pela configuração inicial)
* O cliente pode:
    1. Criar uma ou diversas salas
    2. Entrar/Sair da sala
    3. Definir/Trocar apelido
    4. Entre outros comando (instruções.md)
* Cada sala criada possiu um historico de mensagens


#### Em andamento

* Comunicação entre servidores diferentes de forma descentralizada.

#### Metas
- [ ] **Definir um calendario para a realização das tarefas**
- [ ] Suporte a escalibilidade do numero de servidores
- [ ] Não permitir 2 salas/clientes com o mesmo nome
- [ ] Sincronização das mensagens do cliente.
- [ ] Tolerancia a falhas
- [ ] Ambiente de testes (se possivel)
