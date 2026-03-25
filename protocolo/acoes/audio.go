package acoes

import (
	"slices"
	"strings"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/util"
)

type PostarAudio struct {
	Epoca       uint64
	Autor       crypto.Token
	TipoArquivo string // extensão: .mp3, .wav, .ogg, .flac, .m4a
	Conteudo    crypto.Hash
	Data        time.Time
	Nome        string // nome original do arquivo (opcional)
}

func (p *PostarAudio) ValidarFormato() bool {
	return slices.Contains(TiposAudio, strings.ToLower(p.TipoArquivo))
}

func (p *PostarAudio) FazHash() crypto.Hash {
	return crypto.Hasher(p.Serializa())
}

func (p *PostarAudio) Autoria() crypto.Token {
	return p.Autor
}

func (p *PostarAudio) Serializa() []byte {
	bytes := make([]byte, 0)
	util.PutUint64(p.Epoca, &bytes)
	util.PutToken(p.Autor, &bytes)
	util.PutByte(APostarAudio, &bytes)
	util.PutString(p.TipoArquivo, &bytes)
	util.PutHash(p.Conteudo, &bytes)
	util.PutTime(p.Data, &bytes)
	util.PutString(p.Nome, &bytes)
	return bytes
}

func LeAudio(dados []byte) *PostarAudio {
	acao := PostarAudio{}
	pos := 0
	acao.Epoca, pos = util.ParseUint64(dados, pos)
	acao.Autor, pos = util.ParseToken(dados, pos)
	if dados[pos] != APostarAudio {
		return nil
	}
	pos++
	acao.TipoArquivo, pos = util.ParseString(dados, pos)
	acao.Conteudo, pos = util.ParseHash(dados, pos)
	acao.Data, pos = util.ParseTime(dados, pos)
	if pos < len(dados) {
		// campo Nome opcional: presente em postagens novas, ausente em antigas
		acao.Nome, pos = util.ParseString(dados, pos)
	}
	if pos != len(dados) {
		return nil
	}
	return &acao
}
