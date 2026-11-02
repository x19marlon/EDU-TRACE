package compiler

const (
	entradaHeaderName = "edutrace_entrada.h"
	entradaEnvVar = "EDUTRACE_SALTO"
)

const entradaHeader = `#pragma GCC system_header
#include <cstdio>
#include <cstdlib>
#include <iostream>
#include <string>
namespace edutrace_entrada {
class Salida : public std::streambuf {
public:
	explicit Salida(std::streambuf* destino) : destino_(destino), ultimo_('\n') {}
	void saltar() {
		if (ultimo_ != '\n') {
			destino_->sputc('\n');
			ultimo_ = '\n';
		}
	}
protected:
	int overflow(int c) override {
		if (c == traits_type::eof()) return traits_type::not_eof(c);
		ultimo_ = traits_type::to_char_type(c);
		return destino_->sputc(ultimo_);
	}
	std::streamsize xsputn(const char* s, std::streamsize n) override {
		if (n > 0) ultimo_ = s[n - 1];
		return destino_->sputn(s, n);
	}
	int sync() override { return destino_->pubsync(); }
private:
	std::streambuf* destino_;
	char ultimo_;
};
class Entrada : public std::streambuf {
public:
	explicit Entrada(Salida* salida) : salida_(salida) {}
protected:
	int_type underflow() override {
		std::cout.flush();
		linea_.clear();
		int c;
		while ((c = std::getchar()) != EOF) {
			linea_.push_back(static_cast<char>(c));
			if (c == '\n') break;
		}
		if (linea_.empty()) return traits_type::eof();
		salida_->saltar();
		char* inicio = &linea_[0];
		setg(inicio, inicio, inicio + linea_.size());
		return traits_type::to_int_type(*inicio);
	}
private:
	Salida* salida_;
	std::string linea_;
};
struct Inicio {
	Inicio() {
		if (!std::getenv("EDUTRACE_SALTO")) return;
		Salida* salida = new Salida(std::cout.rdbuf());
		std::cout.rdbuf(salida);
		std::cin.rdbuf(new Entrada(salida));
	}
};
static Inicio inicio;
}
`
