package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/jornal/protocolo/acoes"
	"github.com/freehandle/jornal/protocolo/estado"
)

type InformacaoCabecalho struct {
	Arroba    string
	Erro      string
	NomeMucua string
}

type ViewConvite struct {
	Cabecalho InformacaoCabecalho
	Seed      string
}

type VerPostagem struct {
	Cabecalho  InformacaoCabecalho
	PodeTexto  bool
	PodeImagem bool
	PodeAudio  bool
}

type ItemPost struct {
	Data      string
	Conteudo  string // texto ou "hash.extensão" para arquivos
	TipoTexto bool
	TipoHash  bool
	Nome      string // nome original do arquivo (áudio)
	Autoria   string // @arroba do autor (áudio)
}

type PaginaJornal struct {
	NomeMucua string
	Arroba    string
	Logado    bool
	EhDono    bool
	OptIn     bool
	DataAtual string
	Textos    []ItemPost
	Imagens   []ItemPost
	Audios    []ItemPost
}

type ItemLanding struct {
	Arroba          string
	UltimaData      string
	TiposPublicados string
	ultimaEpoca     uint64
}

type PaginaLanding struct {
	NomeMucua    string
	DataAtual    string
	Jornais      []ItemLanding
	Logado       bool
	ArrobaLogado string
	OptIn        bool
}

func dataFormatada(a *Aplicacao, epoca uint64) string {
	return a.DataDaEpoca(epoca).Format("02/01/2006 15:04")
}

func dataHoje() string {
	diasSemana := []string{"DOMINGO", "SEGUNDA-FEIRA", "TERÇA-FEIRA", "QUARTA-FEIRA", "QUINTA-FEIRA", "SEXTA-FEIRA", "SÁBADO"}
	meses := []string{"JANEIRO", "FEVEREIRO", "MARÇO", "ABRIL", "MAIO", "JUNHO",
		"JULHO", "AGOSTO", "SETEMBRO", "OUTUBRO", "NOVEMBRO", "DEZEMBRO"}
	t := time.Now()
	return fmt.Sprintf("%s — %d DE %s DE %d", diasSemana[t.Weekday()], t.Day(), meses[t.Month()-1], t.Year())
}

func splitURL(path string) []string {
	partes := strings.Split(path, "/")
	result := make([]string, 0, len(partes))
	for _, p := range partes {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (a *Aplicacao) ManejoCredenciais(w http.ResponseWriter, r *http.Request) {
	arroba, _ := a.Gerente.SessionUser(r)
	if arroba != "" {
		http.Redirect(w, r, fmt.Sprintf("%s/jornal/%s", a.NomeMucua, arroba), http.StatusSeeOther)
		return
	}
	view := InformacaoCabecalho{NomeMucua: a.NomeMucua}
	if err := a.templates.ExecuteTemplate(w, "credenciais.html", view); err != nil {
		log.Println(err)
	}
}

func (a *Aplicacao) ManejoSignin(w http.ResponseWriter, r *http.Request) {
	hashEncoded := strings.TrimPrefix(r.URL.Path, "/signin/")
	hash := crypto.DecodeHash(hashEncoded)
	_, convidado := a.Convidar[hash]
	if !convidado && len(a.Convidar) > 0 {
		view := InformacaoCabecalho{Erro: "convite inválido", NomeMucua: a.NomeMucua}
		a.templates.ExecuteTemplate(w, "credenciais.html", view)
		return
	}
	view := ViewConvite{
		Cabecalho: InformacaoCabecalho{NomeMucua: a.NomeMucua},
		Seed:      hashEncoded,
	}
	if err := a.templates.ExecuteTemplate(w, "signin.html", view); err != nil {
		log.Println(err)
	}
}

func (a *Aplicacao) ManejoNovoUsuario(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	arroba := r.FormValue("handle")
	email := r.FormValue("email")
	senha := r.FormValue("password")
	ok := a.Gerente.OnboardSigner(arroba, email, senha)
	aviso := InformacaoCabecalho{NomeMucua: a.NomeMucua}
	if !ok {
		aviso.Erro = "Confira seu email para ativar sua conta ou tente outro arroba."
	}
	if err := a.templates.ExecuteTemplate(w, "credenciais.html", aviso); err != nil {
		log.Println(err)
	}
}

func (a *Aplicacao) ManejoCatraca(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	arroba := r.FormValue("usuario")
	senha := r.FormValue("senha")
	token, ok := a.Indice.ArrobaParaToken[arroba]
	if ok && a.Gerente.Check(token, senha) {
		sessao, err := a.Gerente.CreateSession(arroba)
		if err == nil {
			http.SetCookie(w, sessao)
			http.Redirect(w, r, fmt.Sprintf("%s/jornal/%s", a.NomeMucua, arroba), http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("%s/credenciais", a.NomeMucua), http.StatusSeeOther)
}

func (a *Aplicacao) ManejoSair(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(appName)
	if err == nil {
		if token, ok := a.Gerente.Cookies.Get(cookie.Value); ok {
			a.Gerente.Cookies.Unset(token, cookie.Value)
		}
	}
	http.Redirect(w, r, fmt.Sprintf("%s/credenciais", a.NomeMucua), http.StatusSeeOther)
}

func (a *Aplicacao) ManejoJornal(w http.ResponseWriter, r *http.Request) {
	strToken := a.Autor(r)
	partes := splitURL(r.URL.Path)
	if len(partes) < 2 {
		http.NotFound(w, r)
		return
	}
	arroba := partes[len(partes)-1]
	jornal, ok := a.Indice.ArrobaParaJornal[arroba]
	if !ok {
		http.NotFound(w, r)
		return
	}

	tokenLogado := crypto.TokenFromString(strToken)
	_, optado := a.OptIn[arroba]
	pagina := PaginaJornal{
		NomeMucua: a.NomeMucua,
		Arroba:    arroba,
		Logado:    strToken != "",
		EhDono:    strToken != "" && a.Indice.TokenParaArroba[tokenLogado] == arroba,
		OptIn:     optado,
		DataAtual: dataHoje(),
	}
	fmt.Println(pagina)
	for i := len(jornal.Textos) - 1; i >= 0; i-- {
		t := jornal.Textos[i]
		pagina.Textos = append(pagina.Textos, ItemPost{
			Data:      dataFormatada(a, t.Data),
			Conteudo:  t.Conteudo,
			TipoTexto: true,
		})
	}
	for i := len(jornal.Imagens) - 1; i >= 0; i-- {
		img := jornal.Imagens[i]
		pagina.Imagens = append(pagina.Imagens, ItemPost{
			Data:     dataFormatada(a, img.Data),
			Conteudo: fmt.Sprintf("%s%s", img.Hash.String(), img.Tipo),
			TipoHash: true,
		})
	}
	for i := len(jornal.Audios) - 1; i >= 0; i-- {
		aud := jornal.Audios[i]
		pagina.Audios = append(pagina.Audios, ItemPost{
			Data:     dataFormatada(a, aud.Data),
			Conteudo: fmt.Sprintf("%s%s", aud.Hash.String(), aud.Tipo),
			TipoHash: true,
			Nome:     aud.Nome,
			Autoria:  arroba,
		})
	}

	if err := a.templates.ExecuteTemplate(w, "jornal.html", pagina); err != nil {
		log.Println(err)
	}
}

func (a *Aplicacao) ManejoPostagem(w http.ResponseWriter, r *http.Request) {
	strToken := a.Autor(r)
	token := crypto.TokenFromString(strToken)
	arroba, ok := a.Indice.TokenParaArroba[token]
	if strToken == "" || !ok {
		http.Redirect(w, r, fmt.Sprintf("%s/credenciais", a.NomeMucua), http.StatusSeeOther)
		return
	}
	view := VerPostagem{
		Cabecalho: InformacaoCabecalho{Arroba: arroba, NomeMucua: a.NomeMucua},
	}
	jornal := a.Indice.ArrobaParaJornal[arroba]
	if jornal != nil {
		view.PodeTexto = len(jornal.Textos) == 0 ||
			a.Epoca-jornal.Textos[len(jornal.Textos)-1].Data >= estado.LapsoDia
		view.PodeImagem = len(jornal.Imagens) == 0 ||
			a.Epoca-jornal.Imagens[len(jornal.Imagens)-1].Data >= estado.LapsoDia
		view.PodeAudio = len(jornal.Audios) == 0 ||
			a.Epoca-jornal.Audios[len(jornal.Audios)-1].Data >= estado.LapsoDia
	}
	if err := a.templates.ExecuteTemplate(w, "postagem.html", view); err != nil {
		log.Println(err)
	}
}

func (a *Aplicacao) ManejoPublica(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	strToken := a.Autor(r)
	token := crypto.TokenFromString(strToken)
	arroba, ok := a.Indice.TokenParaArroba[token]
	if strToken == "" || !ok {
		http.Redirect(w, r, fmt.Sprintf("%s/credenciais", a.NomeMucua), http.StatusSeeOther)
		return
	}

	if err := r.ParseMultipartForm(20_000_000); err != nil {
		http.Error(w, "Erro ao processar formulário", http.StatusBadRequest)
		return
	}

	tipo := r.FormValue("tipoConteudo")
	switch tipo {
	case "texto":
		conteudo := r.FormValue("textoConteudo")
		// Se um arquivo .txt ou .md foi enviado, usa o conteúdo do arquivo
		arquivoTexto, cabecalhoTexto, errArq := r.FormFile("arquivoTexto")
		if errArq == nil {
			ext := strings.ToLower(filepath.Ext(cabecalhoTexto.Filename))
			if ext == ".txt" || ext == ".md" {
				bytesTexto, errLer := io.ReadAll(arquivoTexto)
				if errLer != nil {
					http.Error(w, "Erro ao ler arquivo de texto", http.StatusBadRequest)
					return
				}
				conteudo = string(bytesTexto)
			}
		}
		if len([]rune(conteudo)) > 5000 {
			http.Error(w, "texto excede o limite de 5000 caracteres", http.StatusBadRequest)
			return
		}
		acao := &acoes.PostarTexto{
			Epoca:    a.Epoca,
			Autor:    token,
			Conteudo: conteudo,
			Data:     time.Now(),
		}
		if !acao.ValidarFormato() {
			http.Error(w, "texto deve ter entre 1 e 5000 caracteres", http.StatusBadRequest)
			return
		}
		a.Gateway.Encaminha([]acoes.Acao{acao}, token, a.Epoca)

	case "imagem", "audio":
		arquivo, cabecalho, err := r.FormFile("subir")
		if err != nil {
			http.Error(w, "Erro ao receber arquivo", http.StatusBadRequest)
			return
		}
		if cabecalho.Size > 20_000_000 {
			http.Error(w, "Arquivo muito grande (máx 20MB)", http.StatusBadRequest)
			return
		}
		bytes, err := io.ReadAll(arquivo)
		if err != nil {
			http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
			return
		}
		extensao := filepath.Ext(cabecalho.Filename)
		nomeOriginal := strings.TrimSuffix(cabecalho.Filename, extensao)
		hash := crypto.Hasher(bytes)
		nomearquivo := fmt.Sprintf("%s%s", hash.String(), extensao)
		caminho := filepath.Join(a.CaminhoArquivos, nomearquivo)
		if err := os.WriteFile(caminho, bytes, 0644); err != nil {
			http.Error(w, "Erro ao salvar arquivo", http.StatusInternalServerError)
			return
		}
		if tipo == "imagem" {
			acao := &acoes.PostarImagem{
				Epoca:       a.Epoca,
				Autor:       token,
				TipoArquivo: extensao,
				Conteudo:    hash,
				Data:        time.Now(),
			}
			if !acao.ValidarFormato() {
				http.Error(w, "tipo de imagem inválido (use jpg, gif, png, bmp, svg, webp)", http.StatusBadRequest)
				return
			}
			a.Gateway.Encaminha([]acoes.Acao{acao}, token, a.Epoca)
		} else {
			acao := &acoes.PostarAudio{
				Epoca:       a.Epoca,
				Autor:       token,
				TipoArquivo: extensao,
				Conteudo:    hash,
				Data:        time.Now(),
				Nome:        nomeOriginal,
			}
			if !acao.ValidarFormato() {
				http.Error(w, "tipo de áudio inválido (use mp3, wav, ogg, flac, m4a)", http.StatusBadRequest)
				return
			}
			a.Gateway.Encaminha([]acoes.Acao{acao}, token, a.Epoca)
		}

	default:
		http.Error(w, "tipo de conteúdo desconhecido", http.StatusBadRequest)
		return
	}

	time.Sleep(time.Second)
	http.Redirect(w, r, fmt.Sprintf("%s/jornal/%s", a.NomeMucua, arroba), http.StatusSeeOther)
}

// ManejoConteudo serve os arquivos de mídia (imagens e áudios) pelo hash
func (a *Aplicacao) ManejoConteudo(w http.ResponseWriter, r *http.Request) {
	partes := splitURL(r.URL.Path)
	if len(partes) == 0 {
		http.NotFound(w, r)
		return
	}
	arquivo := filepath.Join(a.CaminhoArquivos, partes[len(partes)-1])
	bytes, err := os.ReadFile(arquivo)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(bytes)
}

// ManejoInicio exibe a landing page com jornais atualizados nas últimas 24h
func (a *Aplicacao) ManejoInicio(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "" {
		http.NotFound(w, r)
		return
	}
	strToken := a.Autor(r)
	var arrobaLogado string
	if strToken != "" {
		tokenLogado := crypto.TokenFromString(strToken)
		arrobaLogado = a.Indice.TokenParaArroba[tokenLogado]
	}
	_, optadoLogado := a.OptIn[arrobaLogado]

	var jornais []ItemLanding
	for arroba := range a.OptIn {
		jornal := a.Indice.ArrobaParaJornal[arroba]
		if jornal == nil {
			continue
		}
		var ultimaEpoca uint64
		var tipos []string

		if len(jornal.Textos) > 0 {
			e := jornal.Textos[len(jornal.Textos)-1].Data
			if a.Epoca >= e && a.Epoca-e < estado.LapsoDia {
				tipos = append(tipos, "Texto")
				if e > ultimaEpoca {
					ultimaEpoca = e
				}
			}
		}
		if len(jornal.Imagens) > 0 {
			e := jornal.Imagens[len(jornal.Imagens)-1].Data
			if a.Epoca >= e && a.Epoca-e < estado.LapsoDia {
				tipos = append(tipos, "Imagem")
				if e > ultimaEpoca {
					ultimaEpoca = e
				}
			}
		}
		if len(jornal.Audios) > 0 {
			e := jornal.Audios[len(jornal.Audios)-1].Data
			if a.Epoca >= e && a.Epoca-e < estado.LapsoDia {
				tipos = append(tipos, "Áudio")
				if e > ultimaEpoca {
					ultimaEpoca = e
				}
			}
		}
		if len(tipos) == 0 {
			continue
		}
		jornais = append(jornais, ItemLanding{
			Arroba:          arroba,
			UltimaData:      dataFormatada(a, ultimaEpoca),
			TiposPublicados: strings.Join(tipos, " · "),
			ultimaEpoca:     ultimaEpoca,
		})
	}
	sort.Slice(jornais, func(i, j int) bool {
		return jornais[i].ultimaEpoca > jornais[j].ultimaEpoca
	})

	pagina := PaginaLanding{
		NomeMucua:    a.NomeMucua,
		DataAtual:    dataHoje(),
		Jornais:      jornais,
		Logado:       strToken != "",
		ArrobaLogado: arrobaLogado,
		OptIn:        optadoLogado,
	}
	if err := a.templates.ExecuteTemplate(w, "landing.html", pagina); err != nil {
		log.Println(err)
	}
}

// ManejoOptIn alterna a participação do jornal do usuário na landing page
func (a *Aplicacao) ManejoOptIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	strToken := a.Autor(r)
	token := crypto.TokenFromString(strToken)
	arroba, ok := a.Indice.TokenParaArroba[token]
	if strToken == "" || !ok {
		http.Redirect(w, r, fmt.Sprintf("%s/credenciais", a.NomeMucua), http.StatusSeeOther)
		return
	}
	if _, opted := a.OptIn[arroba]; opted {
		delete(a.OptIn, arroba)
	} else {
		a.OptIn[arroba] = struct{}{}
	}
	if a.CaminhoOptIn != "" {
		SalvarOptIn(a.CaminhoOptIn, a.OptIn)
	}
	http.Redirect(w, r, fmt.Sprintf("%s/jornal/%s", a.NomeMucua, arroba), http.StatusSeeOther)
}
