package app

import (
	"fmt"

	"github.com/freehandle/breeze/crypto"
	"github.com/freehandle/iu/auth"
)

func ContrataGerente(app *Aplicacao, path, senhaGmail, usuarioGmail string, credenciais crypto.PrivateKey) (*auth.SigninManager, error) {
	arqSenhas := fmt.Sprintf("%s/senhas.dat", path)
	senhas := auth.NewFilePasswordManager(arqSenhas)

	mensagens := auth.MessagesTemplates{
		Reset:                    "Para redefinir sua senha, clique no link: %s",
		ResetHeader:              "Redefinição de senha - Jornal",
		Signin:                   "Para entrar no Jornal, clique no link: %s",
		SigninHeader:             "Entrar no Jornal",
		Wellcome:                 "Bem-vindo ao Jornal! Sua conta foi criada.",
		WellcomeHeader:           "Bem-vindo ao Jornal",
		EmailSigninMessage:       "Para entrar sem handle, clique: %s",
		EmailSigninMessageHeader: "Entrar sem handle",
		PasswordMessage:          "Sua nova senha é: %s",
		PasswordMessageHeader:    "Nova senha - Jornal",
		VerifyPOAHeader:          "JORNAL - Confirmação de email",
		VerifyPOA:                "Foi requerida a autorização de uso do seu handle %v para a aplicação %v.\nSe não foi você, ignore esta mensagem.\n\nPara autorizar, clique:\n\n%s",
	}

	var gmail auth.Mailer
	if usuarioGmail == "" {
		gmail = auth.TesteGmail{}
	} else {
		gmail = &auth.SMTPGmail{Password: senhaGmail, From: usuarioGmail}
	}

	carteiro := &auth.SMTPManager{
		Mail:      gmail,
		Token:     credenciais.PublicKey(),
		Templates: mensagens,
	}

	arqCookies := fmt.Sprintf("%s/cookies.dat", path)
	doceria, err := auth.OpenCokieStore(arqCookies)
	if err != nil {
		return nil, err
	}

	gerente := &auth.SigninManager{
		AppName:        appName,
		Passwords:      senhas,
		Cookies:        doceria,
		Mail:           carteiro,
		Granted:        make(map[string]crypto.Token),
		Credentials:    credenciais,
		Members:        app,
		SafeAddress:    "http://localhost:8089",
		SafeAPIAddress: "http://localhost:8090",
		HandleToToken:  make(map[string]crypto.Token),
		TokenToHandle:  make(map[crypto.Token]string),
	}
	return gerente, nil
}
