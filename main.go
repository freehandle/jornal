package main

import (
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"log"
	"os"
	"strings"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/middleware/simple"
	"github.com/freehandle/iu/dev"
	"github.com/freehandle/jornal/app"
	"github.com/freehandle/jornal/indice"
	"github.com/freehandle/jornal/protocolo/estado"
)

var oidKeyEd25519 = asn1.ObjectIdentifier{1, 3, 101, 112}

func carregarOuCriarChave(caminho string) crypto.PrivateKey {
	if dados, err := os.ReadFile(caminho); err == nil {
		pk, err := crypto.ParsePEMPrivateKey(dados)
		if err == nil {
			log.Printf("chave carregada de %s (token: %s)", caminho, pk.PublicKey())
			return pk
		}
		log.Printf("aviso: falha ao interpretar %s: %v — criando nova chave", caminho, err)
	}

	_, pk := crypto.RandomAsymetricKey()

	var seed [32]byte
	copy(seed[:], pk[:32])
	seedASN1, err := asn1.Marshal(seed[:])
	if err != nil {
		log.Fatalf("erro ao codificar semente: %v", err)
	}
	type pkcs8 struct {
		Version    int
		Algo       pkix.AlgorithmIdentifier
		PrivateKey []byte
	}
	pkcs8Key := pkcs8{
		Version:    0,
		Algo:       pkix.AlgorithmIdentifier{Algorithm: oidKeyEd25519},
		PrivateKey: seedASN1,
	}
	der, err := asn1.Marshal(pkcs8Key)
	if err != nil {
		log.Fatalf("erro ao codificar chave PKCS8: %v", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	if err := os.WriteFile(caminho, pem.EncodeToMemory(block), 0600); err != nil {
		log.Fatalf("erro ao gravar %s: %v", caminho, err)
	}
	log.Printf("nova chave criada e gravada em %s (token: %s)", caminho, pk.PublicKey())
	return pk
}

func main() {
	// lê variáveis de ambiente opcionais
	var senhaEmail string
	//caminhoBlocos := "/home/lienko/setembro/handles/cmd/proxy-handles"
	caminhoBlocos := "."

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "SENHA_EMAIL=") {
			senhaEmail, _ = strings.CutPrefix(env, "SENHA_EMAIL=")
		} else if strings.HasPrefix(env, "CAMINHO_BLOCOS=") {
			caminhoBlocos, _ = strings.CutPrefix(env, "CAMINHO_BLOCOS=")
		}
	}

	pk := carregarOuCriarChave("chave.pem")
	token := pk.PublicKey()

	// token do nó Breeze local (gateway)
	breezeToken := crypto.TokenFromString("91ad274d06c4be307a332a0e59449ad25ae2c65e4ad5a8f0af87067ac2fc3a54")

	ctx := context.Background()

	aplicacao := app.NovaAplicacaoVazia()

	_, err := dev.Start(ctx, caminhoBlocos)
	if err != nil {
		log.Fatalf("erro ao iniciar stack de desenvolvimento: %v", err)
	}

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
	//aplicacao.NomeMucua = "/jornal"
	aplicacao.CaminhoArquivos = "/home/lari/conteudojornal/"
	aplicacao.CaminhoOptIn = "./optin.dat"
	aplicacao.OptIn = app.CarregarOptIn("./optin.dat")
	aplicacao.Gateway = app.PorteiraDeCanal(sender, pk)
	if senhaEmail == "" {
		aplicacao.Gerente, err = app.ContrataGerente(aplicacao, ".", "", "", pk)
	} else {
		aplicacao.Gerente, err = app.ContrataGerente(aplicacao, ".", senhaEmail, "arrobaslivres@gmail.com", pk)
	}
	if err != nil {
		log.Fatal(err)
	}
	fim := make(chan error, 1)
	app.NovaMucua(ctx, aplicacao, 8030, "./app")
	err = <-fim
}
