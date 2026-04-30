package app

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/util"
	"github.com/freehandle/handles/attorney"
	"github.com/freehandle/iu/auth"
	"github.com/freehandle/jornal/indice"
	"github.com/freehandle/jornal/protocolo/estado"
)

const appName = "JORNAL"

var arquivosTemplate = []string{
	"credenciais", "signin", "jornal", "postagem", "landing",
}

type Aplicacao struct {
	Epoca           uint64
	Credenciais     crypto.PrivateKey
	Token           crypto.Token
	Gateway         *Porteira
	Novidades       chan []byte
	Eventos         chan auth.IdentityEvent
	Estado          *estado.Estado
	Indice          *indice.Indice
	templates       *template.Template
	GenesisTime     time.Time
	Intervalo       time.Duration
	NomeMucua       string
	Convidar        map[crypto.Hash]struct{}
	Gerente         *auth.SigninManager
	CaminhoArquivos string
	OptIn           map[string]struct{}
	CaminhoOptIn    string
}

func (p *Aplicacao) DataDaEpoca(epoca uint64) time.Time {
	return p.GenesisTime.Add(time.Duration(epoca) * p.Intervalo)
}

func (p *Aplicacao) EpocaDaData(data time.Time) uint64 {
	return uint64(data.Sub(p.GenesisTime) / p.Intervalo)
}

// Rodar processa o stream de ações recebidas do Breeze em loop.
// Consome dois canais: Novidades (bytes brutos: épocas + voids do protocolo
// jornal) e Eventos (ações de identidade já parseadas pelo Gerente).
func (p *Aplicacao) Rodar(ctx context.Context) {
	validador := p.Estado.Validator()
	for {
		select {
		case <-ctx.Done():
			log.Println("Aplicacao.Rodar: contexto encerrado")
			return
		case ev := <-p.Eventos:
			switch ev.Action {
			case auth.Join:
				// usuário registrou handle no protocolo Axé: indexa a arroba
				p.Indice.IncorporaAutor(ev.Handle, ev.Token)
			case auth.Grant:
				// signin do usuário neste app — Indice cria o jornal lazy quando
				// chegar a primeira postagem; nada a fazer aqui por enquanto.
			case auth.Revoke:
				// signout do usuário — posts históricos permanecem; futuras
				// ações já serão recusadas pelo validador da chain.
			}
		case novidade := <-p.Novidades:
			if len(novidade) == 0 {
				continue
			}
			if novidade[0] == 0 {
				// atualização de época: incorpora mutações e reinicia o validador
				if len(novidade) >= 9 {
					epoca, _ := util.ParseUint64(novidade, 1)
					mutacoes := validador.Mutations()
					p.Estado.Incorporate(mutacoes)
					validador = p.Estado.Validator()
					validador.Mutacoes.Epoca = epoca
					p.Epoca = epoca
					p.Estado.Epoca = epoca
				}
				continue
			}
			acao := novidade[1:]
			// só voids do protocolo jornal [1, 3, 0, 0] interessam aqui;
			// JoinNetwork/Grant/Revoke chegam pelo canal Eventos.
			if attorney.Kind(acao) == attorney.VoidType &&
				len(acao) > 13 && acao[10] == 1 && acao[11] == 3 && acao[12] == 0 && acao[13] == 0 {
				if a := BreezeParaJornal(acao); validador.Validate(a) {
					p.Indice.IncorporaAcao(a)
				}
			}
		}
	}
}

func NovaAplicacaoVazia() *Aplicacao {
	files := make([]string, len(arquivosTemplate))
	for n, file := range arquivosTemplate {
		files[n] = fmt.Sprintf("./app/templates/%s.html", file)
	}
	t, err := template.ParseFiles(files...)
	if err != nil {
		log.Fatal(err)
	}
	return &Aplicacao{
		Convidar:    make(map[crypto.Hash]struct{}),
		OptIn:       make(map[string]struct{}),
		templates:   t,
		GenesisTime: time.Now(),
	}
}

// Invite e AppName/AttorneyToken implementam a interface auth.Associater
func (p *Aplicacao) Invite(handle string, token crypto.Token) error {
	return nil
}

func (p *Aplicacao) AppName() string {
	return appName
}

func (p *Aplicacao) AttorneyToken() crypto.Token {
	return p.Token
}

// Autor retorna o token hex do usuário logado, ou string vazia se não há sessão
func (p *Aplicacao) Autor(r *http.Request) string {
	cookie, err := r.Cookie(appName)
	if err == nil {
		if token, ok := p.Gerente.Cookies.Get(cookie.Value); ok {
			return token.String()
		}
	}
	return ""
}
