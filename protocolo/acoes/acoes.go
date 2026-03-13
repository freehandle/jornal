package acoes

import "github.com/freehandle/breeze/crypto"

// byte identificador de cada tipo de ação do protocolo jornal
const (
	APostarTexto  byte = iota
	APostarImagem
	APostarAudio
	APostarErro
)

var TiposImagens = []string{".jpg", ".gif", ".png", ".bmp", ".svg", ".webp"}
var TiposAudio = []string{".mp3", ".wav", ".ogg", ".flac", ".m4a"}

type Acao interface {
	Serializa() []byte
	Autoria() crypto.Token
	FazHash() crypto.Hash
}

// retorna o byte do tipo de ação a partir dos bytes serializados
func TipoDeAcao(dados []byte) byte {
	if len(dados) < 8+crypto.TokenSize+1 {
		return APostarErro
	}
	byteAcao := dados[8+crypto.TokenSize]
	if byteAcao >= APostarErro {
		return APostarErro
	}
	return byteAcao
}
