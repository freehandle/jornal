package acoes

import (
	"time"
	"unicode/utf8"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/util"
)

type PostarTexto struct {
	Epoca    uint64
	Autor    crypto.Token
	Conteudo string // 1 a 5000 caracteres
	Data     time.Time
}

func (p *PostarTexto) ValidarFormato() bool {
	n := utf8.RuneCountInString(p.Conteudo)
	return n >= 1 && n <= 5000
}

func (p *PostarTexto) FazHash() crypto.Hash {
	return crypto.Hasher(p.Serializa())
}

func (p *PostarTexto) Autoria() crypto.Token {
	return p.Autor
}

func (p *PostarTexto) Serializa() []byte {
	bytes := make([]byte, 0)
	util.PutUint64(p.Epoca, &bytes)
	util.PutToken(p.Autor, &bytes)
	util.PutByte(APostarTexto, &bytes)
	util.PutString(p.Conteudo, &bytes)
	util.PutTime(p.Data, &bytes)
	return bytes
}

func LeTexto(dados []byte) *PostarTexto {
	acao := PostarTexto{}
	pos := 0
	acao.Epoca, pos = util.ParseUint64(dados, pos)
	acao.Autor, pos = util.ParseToken(dados, pos)
	if dados[pos] != APostarTexto {
		return nil
	}
	pos++
	acao.Conteudo, pos = util.ParseString(dados, pos)
	acao.Data, pos = util.ParseTime(dados, pos)
	if pos != len(dados) {
		return nil
	}
	return &acao
}
