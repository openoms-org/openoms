"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { BookOpen, HelpCircle, MessageCircle, Bug, Mail } from "lucide-react";
import Link from "next/link";

const helpLinks = [
  {
    title: "Poradnik użytkownika",
    description: "Krok po kroku: od rejestracji do pierwszego zamówienia.",
    href: "https://github.com/openoms-org/openoms/blob/main/docs/poradnik-uzytkownika.md",
    icon: BookOpen,
    external: true,
  },
  {
    title: "FAQ — Pytania i odpowiedzi",
    description: "Najczęściej zadawane pytania przez sprzedawców.",
    href: "https://github.com/openoms-org/openoms/blob/main/docs/faq-sprzedawcy.md",
    icon: HelpCircle,
    external: true,
  },
  {
    title: "Społeczność Discord",
    description: "Dołącz do społeczności, zadawaj pytania, dziel się pomysłami.",
    href: "https://discord.gg/openoms",
    icon: MessageCircle,
    external: true,
  },
  {
    title: "Zgłoś problem",
    description: "Znalazłeś błąd? Otwórz issue na GitHubie.",
    href: "https://github.com/openoms-org/openoms/issues",
    icon: Bug,
    external: true,
  },
];

export default function HelpPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Pomoc</h1>
        <p className="text-muted-foreground mt-1">Zasoby i wsparcie dla użytkowników OpenOMS.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {helpLinks.map((link) => (
          <Link
            key={link.title}
            href={link.href}
            target={link.external ? "_blank" : undefined}
            rel={link.external ? "noopener noreferrer" : undefined}
          >
            <Card className="h-full hover:bg-accent/50 transition-colors cursor-pointer">
              <CardHeader className="flex flex-row items-center gap-3 pb-2">
                <link.icon className="h-5 w-5 text-primary" />
                <CardTitle className="text-base">{link.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{link.description}</p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <Card>
        <CardContent className="flex items-center gap-3 pt-6">
          <Mail className="h-5 w-5 text-muted-foreground" />
          <p className="text-sm">
            Potrzebujesz bezpośredniej pomocy?{" "}
            <a href="mailto:kontakt@openoms.pl" className="text-primary underline">
              kontakt@openoms.pl
            </a>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
