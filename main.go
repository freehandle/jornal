package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/middleware/simple"
	"github.com/freehandle/jornal/app"
	"github.com/freehandle/jornal/indice"
	"github.com/freehandle/jornal/protocolo/estado"
)

func main() {
	// lê variáveis de ambiente opcionais
	var senhaEmail string
	caminhoBlocos := "/home/lienko/setembro/handles/cmd/proxy-handles"

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "SENHA_EMAIL=") {
			senhaEmail, _ = strings.CutPrefix(env, "SENHA_EMAIL=")
		} else if strings.HasPrefix(env, "CAMINHO_BLOCOS=") {
			caminhoBlocos, _ = strings.CutPrefix(env, "CAMINHO_BLOCOS=")
		}
	}

	// chave privada do nó jornal — substitua pela sua chave real
	pk := crypto.PrivateKeyFromString("e18e6528bd958000e51553f1828456c96509a3daa595421e24890d3153962297bb46f0c6a41ffc8ca179f3429d2584f103f66e540e21a197a45295ca8aa045de")
	token := pk.PublicKey()

	// token do nó Breeze local (gateway)
	breezeToken := crypto.TokenFromString("91ad274d06c4be307a332a0e59449ad25ae2c65e4ad5a8f0af87067ac2fc3a54")

	ctx := context.Background()

	aplicacao := app.NovaAplicacaoVazia()

	// fonte de ações: lê blocos do proxy de handles e extrai ações individuais
	novidades := simple.DissociateActions(ctx, simple.NewBlockReader(ctx, caminhoBlocos, "blocos", time.Second))

	// gateway local para envio de ações ao Breeze
	sender, err := simple.Gateway(ctx, 7000, breezeToken, pk)
	if err != nil {
		log.Fatalf("erro ao criar gateway: %v", err)
	}

	aplicacao.Credenciais = pk
	aplicacao.Token = token
	aplicacao.Novidades = novidades
	aplicacao.Estado = estado.Genesis(0)
	aplicacao.Indice = indice.NovoIndice()
	aplicacao.GenesisTime = time.Date(2025, time.September, 14, 15, 10, 10, 0, time.UTC)
	aplicacao.Intervalo = time.Second
	aplicacao.Gateway = app.PorteiraDeCanal(sender, pk)
	aplicacao.NomeMucua = ""
	aplicacao.CaminhoArquivos = "/home/lienko/setembro/arquivosjornal/"
	aplicacao.CaminhoOptIn = "./optin.dat"
	aplicacao.OptIn = app.CarregarOptIn("./optin.dat")

	if senhaEmail == "" {
		aplicacao.Gerente, err = app.ContrataGerente(aplicacao, ".", "", "", pk)
	} else {
		aplicacao.Gerente, err = app.ContrataGerente(aplicacao, ".", senhaEmail, "arrobaslivres@gmail.com", pk)
	}
	if err != nil {
		log.Fatal(err)
	}

	fim := make(chan error, 1)
	app.NovaMucua(ctx, aplicacao, 8080, "./app")
	err = <-fim
}
