package indice

import (
	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/jornal/protocolo/acoes"
)

type Jornal struct {
	Textos  []*ConteudoData
	Imagens []*HashData
	Audios  []*HashData
}

type ConteudoData struct {
	Conteudo string
	Data     uint64
}

type HashData struct {
	Hash crypto.Hash
	Data uint64
	Tipo string
	Nome string // nome original do arquivo (vazio em postagens antigas)
}

func NovoIndice() *Indice {
	return &Indice{
		ArrobaParaToken:  make(map[string]crypto.Token),
		TokenParaArroba:  make(map[crypto.Token]string),
		ArrobaParaJornal: make(map[string]*Jornal),
	}
}

type Indice struct {
	ArrobaParaToken  map[string]crypto.Token
	TokenParaArroba  map[crypto.Token]string
	ArrobaParaJornal map[string]*Jornal
}

func (i *Indice) IncorporaAutor(arroba string, token crypto.Token) {
	i.ArrobaParaToken[arroba] = token
	i.TokenParaArroba[token] = arroba
}

func (i *Indice) jornal(arroba string) *Jornal {
	j, ok := i.ArrobaParaJornal[arroba]
	if !ok {
		j = &Jornal{}
		i.ArrobaParaJornal[arroba] = j
	}
	return j
}

func (i *Indice) IncorporaAcao(dados []byte) {
	tipo := acoes.TipoDeAcao(dados)
	switch tipo {
	case acoes.APostarTexto:
		if acao := acoes.LeTexto(dados); acao != nil {
			arroba := i.TokenParaArroba[acao.Autor]
			jornal := i.jornal(arroba)
			jornal.Textos = append(jornal.Textos, &ConteudoData{
				Conteudo: acao.Conteudo,
				Data:     acao.Epoca,
			})
		}
	case acoes.APostarImagem:
		if acao := acoes.LeImagem(dados); acao != nil {
			arroba := i.TokenParaArroba[acao.Autor]
			jornal := i.jornal(arroba)
			jornal.Imagens = append(jornal.Imagens, &HashData{
				Hash: acao.Conteudo,
				Data: acao.Epoca,
				Tipo: acao.TipoArquivo,
			})
		}
	case acoes.APostarAudio:
		if acao := acoes.LeAudio(dados); acao != nil {
			arroba := i.TokenParaArroba[acao.Autor]
			jornal := i.jornal(arroba)
			jornal.Audios = append(jornal.Audios, &HashData{
				Hash: acao.Conteudo,
				Data: acao.Epoca,
				Tipo: acao.TipoArquivo,
				Nome: acao.Nome,
			})
		}
	}
}
