package services

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// ============================================================
// Pengirim email lewat SMTP
//
// Saat ini dipakai dengan Gmail (smtp.gmail.com:587 + App Password), tapi
// tidak ada yang khusus Gmail — ganti SMTP_HOST/PORT untuk layanan lain.
// Semua dibaca dari env saat dipakai, bukan saat package di-load, karena
// godotenv.Load() baru dipanggil di main() (lihat AttachmentStorageRoot).
// ============================================================

type mailerConfig struct {
	host      string
	port      string
	username  string
	password  string
	fromEmail string
	fromName  string
}

func loadMailerConfig() mailerConfig {
	cfg := mailerConfig{
		host:      strings.TrimSpace(os.Getenv("SMTP_HOST")),
		port:      strings.TrimSpace(os.Getenv("SMTP_PORT")),
		username:  strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		password:  os.Getenv("SMTP_PASSWORD"),
		fromEmail: strings.TrimSpace(os.Getenv("SMTP_FROM_EMAIL")),
		fromName:  strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
	}
	if cfg.host == "" {
		cfg.host = "smtp.gmail.com"
	}
	if cfg.port == "" {
		cfg.port = "587"
	}
	// Gmail menolak alamat From yang bukan milik akun login, jadi default-nya
	// disamakan dengan username.
	if cfg.fromEmail == "" {
		cfg.fromEmail = cfg.username
	}
	if cfg.fromName == "" {
		cfg.fromName = "Asset Management"
	}
	return cfg
}

// MailerConfigured — false kalau kredensial SMTP belum diisi di .env.
func MailerConfigured() bool {
	cfg := loadMailerConfig()
	return cfg.username != "" && cfg.password != ""
}

type outgoingMail struct {
	to       []string
	cc       []string
	subject  string
	htmlBody string
}

const smtpTimeout = 20 * time.Second

func sendMail(m outgoingMail) error {
	cfg := loadMailerConfig()
	if cfg.username == "" || cfg.password == "" {
		return errors.New("SMTP belum dikonfigurasi (SMTP_USERNAME / SMTP_PASSWORD kosong di .env)")
	}
	if len(m.to) == 0 {
		return errors.New("penerima email kosong")
	}

	// Alamat sudah divalidasi di binding, tapi karena nilainya masuk ke header
	// mentah, pastikan sekali lagi tidak ada CR/LF (header injection).
	for _, addr := range append(append([]string{}, m.to...), m.cc...) {
		if strings.ContainsAny(addr, "\r\n") {
			return fmt.Errorf("alamat email tidak valid: %q", addr)
		}
	}

	message, err := buildMimeMessage(cfg, m)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(cfg.host, cfg.port)
	tlsConfig := &tls.Config{ServerName: cfg.host}

	var conn net.Conn
	dialer := &net.Dialer{Timeout: smtpTimeout}
	if cfg.port == "465" {
		// 465 = implicit TLS (SMTPS)
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("gagal terhubung ke SMTP %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(2 * smtpTimeout))

	client, err := smtp.NewClient(conn, cfg.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("gagal membuka sesi SMTP: %w", err)
	}
	defer client.Close()

	if cfg.port != "465" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("server SMTP tidak mendukung STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS gagal: %w", err)
		}
	}

	if err := client.Auth(smtp.PlainAuth("", cfg.username, cfg.password, cfg.host)); err != nil {
		return fmt.Errorf("login SMTP gagal (cek SMTP_USERNAME / App Password): %w", err)
	}

	if err := client.Mail(cfg.fromEmail); err != nil {
		return fmt.Errorf("pengirim ditolak: %w", err)
	}
	for _, rcpt := range append(append([]string{}, m.to...), m.cc...) {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("penerima %s ditolak: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("gagal mengirim isi email: %w", err)
	}
	if _, err := w.Write(message); err != nil {
		w.Close()
		return fmt.Errorf("gagal mengirim isi email: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("server SMTP menolak email: %w", err)
	}

	return client.Quit()
}

func buildMimeMessage(cfg mailerConfig, m outgoingMail) ([]byte, error) {
	from := mail.Address{Name: cfg.fromName, Address: cfg.fromEmail}

	// subject berasal dari template yang diisi admin — buang baris baru
	subject := strings.NewReplacer("\r", " ", "\n", " ").Replace(m.subject)

	domain := "localhost"
	if at := strings.LastIndex(cfg.fromEmail, "@"); at >= 0 {
		domain = cfg.fromEmail[at+1:]
	}

	var buf bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }

	header("From", from.String())
	header("To", strings.Join(m.to, ", "))
	if len(m.cc) > 0 {
		header("Cc", strings.Join(m.cc, ", "))
	}
	header("Subject", mime.QEncoding.Encode("utf-8", subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", randomToken(), domain))
	header("MIME-Version", "1.0")
	header("Content-Type", `text/html; charset="UTF-8"`)
	header("Content-Transfer-Encoding", "base64")
	buf.WriteString("\r\n")

	encoded := base64.StdEncoding.EncodeToString([]byte(m.htmlBody))
	for len(encoded) > 76 {
		buf.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	buf.WriteString(encoded + "\r\n")

	return buf.Bytes(), nil
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
