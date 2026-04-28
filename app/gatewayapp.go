package app

import (
	"log"

	"github.com/freehandle/breeze/crypto"
	breeze "github.com/freehandle/breeze/protocol/actions"
	"github.com/freehandle/breeze/socket"
	"github.com/freehandle/breeze/util"
	"github.com/freehandle/handles/attorney"
	"github.com/freehandle/iu/auth"
	"github.com/freehandle/jornal/protocolo/acoes"
)

// tail de assinatura que o Breeze acrescenta à ação: 2 assinaturas + 2 tokens + 1 uint64
const finalBreeze = 2*crypto.SignatureSize + 2*crypto.TokenSize + 8

type PorteiraLocal struct {
	canal chan []byte
}

type PorteiraRemota struct {
	Conexao *socket.SignedConnection
}

func (p *PorteiraRemota) Send(data []byte) {
	data = append([]byte{0}, data...)
	p.Conexao.Send(data)
}

func PorteiraInternet(conexao *socket.SignedConnection, credenciais crypto.PrivateKey, gerente *auth.SigninManager) *Porteira {
	return &Porteira{portao: &PorteiraRemota{Conexao: conexao}, credenciais: credenciais}
}

func (p *PorteiraLocal) Send(data []byte) {
	p.canal <- data
}

type Portao interface {
	Send([]byte)
}

func PorteiraDeCanal(canal chan []byte, credenciais crypto.PrivateKey) *Porteira {
	return &Porteira{portao: &PorteiraLocal{canal: canal}, credenciais: credenciais}
}

type Porteira struct {
	portao      Portao
	credenciais crypto.PrivateKey
}

// BreezeParaJornal extrai o conteúdo de uma ação jornal embutida num void do Breeze.
// O layout do void é: [0, IVoid][época 8b][código protocolo 4b][VoidType][conteúdo][tail]
func BreezeParaJornal(action []byte) []byte {
	if len(action) <= finalBreeze+15 {
		return nil
	}
	bytes := make([]byte, 8+len(action)-finalBreeze-15)
	copy(bytes[0:8], action[2:10])                      // época
	copy(bytes[8:], action[15:len(action)-finalBreeze]) // conteúdo sem cabeçalho e sem tail
	return bytes
}

// JornalParaBreeze empacota uma ação jornal serializada como void do Breeze,
// usando o código de protocolo [1, 3, 0, 0].
func JornalParaBreeze(action []byte, epoch uint64) []byte {
	if action == nil {
		log.Print("PANIC BUG: JornalParaBreeze chamado com acao nula")
		return nil
	}
	bytes := []byte{0, breeze.IVoid}
	util.PutUint64(epoch, &bytes)
	bytes = append(bytes, 1, 13, 0, 0, attorney.VoidType) // código do protocolo jornal
	bytes = append(bytes, action[8:]...)                  // conteúdo (pula época que já está no cabeçalho)
	return bytes
}

func (p *Porteira) Encaminha(all []acoes.Acao, autoria crypto.Token, epoca uint64) {
	for _, action := range all {
		dressed := p.TravesteAcao(action, autoria, epoca)
		if dressed != nil {
			p.portao.Send(dressed)
		}
	}
}

// TravesteAcao empacota a ação no formato Breeze com assinaturas do procurador e da fee
func (p *Porteira) TravesteAcao(action acoes.Acao, autoria crypto.Token, epoca uint64) []byte {
	bytes := JornalParaBreeze(action.Serializa(), epoca)
	if bytes == nil {
		return nil
	}
	// assinatura do procurador
	util.PutToken(p.credenciais.PublicKey(), &bytes)
	signature := p.credenciais.Sign(bytes)
	util.PutSignature(signature, &bytes)
	// assinatura da fee (fee = 0)
	util.PutToken(p.credenciais.PublicKey(), &bytes)
	util.PutUint64(0, &bytes)
	signature = p.credenciais.Sign(bytes)
	util.PutSignature(signature, &bytes)
	return bytes
}
