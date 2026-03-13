package estado

import (
	"sync"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/breeze/middleware/social"
	"github.com/freehandle/jornal/protocolo/acoes"
)

// cooldown de 1 época (para testes)
const LapsoDia = 24 * 60 * 60

type Postagem struct {
	Token crypto.Token
	Tipo  byte
}

func Genesis(epoch uint64) *Estado {
	return &Estado{
		mu:                sync.Mutex{},
		Epoca:             epoch,
		UltimaAtualizacao: make(map[Postagem]uint64),
	}
}

type Estado struct {
	mu                sync.Mutex
	Epoca             uint64
	UltimaAtualizacao map[Postagem]uint64
}

type EstadoMutante struct {
	Original *Estado
	Mutacoes *Mutacoes
}

func (e *EstadoMutante) Mutations() *Mutacoes {
	return e.Mutacoes
}

func (e *EstadoMutante) ultimaPublicacao(token crypto.Token, tipo byte) uint64 {
	postagem := Postagem{Token: token, Tipo: tipo}
	if ultima, ok := e.Mutacoes.Atualizacoes[postagem]; ok {
		return ultima
	}
	if ultima, ok := e.Original.UltimaAtualizacao[postagem]; ok {
		return ultima
	}
	return 0
}

func (e *EstadoMutante) Validate(dados []byte) bool {
	tipo := acoes.TipoDeAcao(dados)
	switch tipo {
	case acoes.APostarTexto:
		acao := acoes.LeTexto(dados)
		if acao == nil || !acao.ValidarFormato() {
			return false
		}
		ultima := e.ultimaPublicacao(acao.Autor, acoes.APostarTexto)
		if ultima > 0 && acao.Epoca < ultima+LapsoDia {
			return false
		}
		if acao.Epoca <= e.Mutacoes.Epoca {
			e.Mutacoes.Atualizacoes[Postagem{Token: acao.Autor, Tipo: acoes.APostarTexto}] = acao.Epoca
			return true
		}
		return false

	case acoes.APostarImagem:
		acao := acoes.LeImagem(dados)
		if acao == nil || !acao.ValidarFormato() {
			return false
		}
		ultima := e.ultimaPublicacao(acao.Autor, acoes.APostarImagem)
		if ultima > 0 && acao.Epoca < ultima+LapsoDia {
			return false
		}
		if acao.Epoca <= e.Mutacoes.Epoca {
			e.Mutacoes.Atualizacoes[Postagem{Token: acao.Autor, Tipo: acoes.APostarImagem}] = acao.Epoca
			return true
		}
		return false

	case acoes.APostarAudio:
		acao := acoes.LeAudio(dados)
		if acao == nil || !acao.ValidarFormato() {
			return false
		}
		ultima := e.ultimaPublicacao(acao.Autor, acoes.APostarAudio)
		if ultima > 0 && acao.Epoca < ultima+LapsoDia {
			return false
		}
		if acao.Epoca <= e.Mutacoes.Epoca {
			e.Mutacoes.Atualizacoes[Postagem{Token: acao.Autor, Tipo: acoes.APostarAudio}] = acao.Epoca
			return true
		}
		return false

	default:
		return false
	}
}

func (e *Estado) Validator(mutacoes ...*Mutacoes) *EstadoMutante {
	validador := EstadoMutante{Original: e}
	if len(mutacoes) == 0 {
		validador.Mutacoes = &Mutacoes{
			Epoca:        e.Epoca + 1,
			Atualizacoes: make(map[Postagem]uint64),
		}
	} else if len(mutacoes) == 1 {
		validador.Mutacoes = mutacoes[0]
	} else {
		validador.Mutacoes = mutacoes[0].Merge(mutacoes[1:]...)
	}
	return &validador
}

func (e *Estado) Incorporate(mutacoes *Mutacoes) {
	e.Epoca = mutacoes.Epoca
	for k, v := range mutacoes.Atualizacoes {
		e.UltimaAtualizacao[k] = v
	}
}

func (e *Estado) Shutdown() {}

func (e *Estado) Checksum() crypto.Hash {
	return crypto.ZeroHash
}

func (e *Estado) Clone() chan social.Stateful[*Mutacoes, *EstadoMutante] {
	e.mu.Lock()
	defer e.mu.Unlock()
	clone := Estado{
		Epoca:             e.Epoca,
		UltimaAtualizacao: make(map[Postagem]uint64),
	}
	for k, v := range e.UltimaAtualizacao {
		clone.UltimaAtualizacao[k] = v
	}
	resposta := make(chan social.Stateful[*Mutacoes, *EstadoMutante], 2)
	resposta <- &clone
	return resposta
}

func (e *Estado) Serialize() []byte {
	return nil
}

type Mutacoes struct {
	Epoca        uint64
	Atualizacoes map[Postagem]uint64
}

func (m *Mutacoes) Merge(mutacoes ...*Mutacoes) *Mutacoes {
	merged := &Mutacoes{
		Atualizacoes: make(map[Postagem]uint64),
	}
	for _, mu := range append([]*Mutacoes{m}, mutacoes...) {
		for k, v := range mu.Atualizacoes {
			if at, ok := merged.Atualizacoes[k]; !ok || v > at {
				merged.Atualizacoes[k] = v
			}
		}
	}
	return merged
}
