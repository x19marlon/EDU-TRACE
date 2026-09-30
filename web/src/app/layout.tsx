import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "EDU-TRACE — Compilador C++",
  description: "Plataforma de apoyo para Introducción a la Programación",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="es">
      <body className="bg-gray-950 text-white antialiased">
        {children}
      </body>
    </html>
  );
}
