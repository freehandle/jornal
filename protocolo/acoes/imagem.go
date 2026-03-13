package acoes

import (
	"slices"
	"strings"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/util"
)

type PostarImagem struct {
	Epoca       uint64
	Autor       crypto.Token
	TipoArquivo string // extensão: .jpg, .gif, .png, .bmp, .svg, .webp
	Conteudo    crypto.Hash
	Data        time.Time
}

func (p *PostarImagem) ValidarFormato() bool {
	return slices.Contains(TiposImagens, strings.ToLower(p.TipoArquivo))
}

func (p *PostarImagem) FazHash() crypto.Hash {
	return crypto.Hasher(p.Serializa())
}

func (p *PostarImagem) Autoria() crypto.Token {
	return p.Autor
}

func (p *PostarImagem) Serializa() []byte {
	bytes := make([]byte, 0)
	util.PutUint64(p.Epoca, &bytes)
	util.PutToken(p.Autor, &bytes)
	util.PutByte(APostarImagem, &bytes)
	util.PutString(p.TipoArquivo, &bytes)
	util.PutHash(p.Conteudo, &bytes)
	util.PutTime(p.Data, &bytes)
	return bytes
}

func LeImagem(dados []byte) *PostarImagem {
	acao := PostarImagem{}
	pos := 0
	acao.Epoca, pos = util.ParseUint64(dados, pos)
	acao.Autor, pos = util.ParseToken(dados, pos)
	if dados[pos] != APostarImagem {
		return nil
	}
	pos++
	acao.TipoArquivo, pos = util.ParseString(dados, pos)
	acao.Conteudo, pos = util.ParseHash(dados, pos)
	acao.Data, pos = util.ParseTime(dados, pos)
	if pos != len(dados) {
		return nil
	}
	return &acao
}
