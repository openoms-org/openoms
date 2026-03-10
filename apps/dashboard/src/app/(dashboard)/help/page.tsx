"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { BookOpen, HelpCircle, MessageCircle, Bug, Mail } from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";


export default function HelpPage() {
  const t = useTranslations("help");

  const helpLinks = [
    {
      titleKey: "userGuide",
      descriptionKey: "userGuideDescription",
      href: "https://github.com/openoms-org/openoms/blob/main/docs/poradnik-uzytkownika.md",
      icon: BookOpen,
      external: true,
    },
    {
      titleKey: "faq",
      descriptionKey: "faqDescription",
      href: "https://github.com/openoms-org/openoms/blob/main/docs/faq-sprzedawcy.md",
      icon: HelpCircle,
      external: true,
    },
    {
      titleKey: "discordCommunity",
      descriptionKey: "discordDescription",
      href: "https://discord.gg/openoms",
      icon: MessageCircle,
      external: true,
    },
    {
      titleKey: "reportBug",
      descriptionKey: "reportBugDescription",
      href: "https://github.com/openoms-org/openoms/issues",
      icon: Bug,
      external: true,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="text-muted-foreground mt-1">{t("subtitle")}</p>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {helpLinks.map((link) => (
          <Link
            key={link.titleKey}
            href={link.href}
            target={link.external ? "_blank" : undefined}
            rel={link.external ? "noopener noreferrer" : undefined}
          >
            <Card className="h-full hover:bg-accent/50 transition-colors cursor-pointer">
              <CardHeader className="flex flex-row items-center gap-3 pb-2">
                <link.icon className="h-5 w-5 text-primary" />
                <CardTitle className="text-base">{t(link.titleKey)}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{t(link.descriptionKey)}</p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <Card>
        <CardContent className="flex items-center gap-3 pt-6">
          <Mail className="h-5 w-5 text-muted-foreground" />
          <p className="text-sm">
            {t("needDirectHelp")}{" "}
            <a href="mailto:kontakt@openoms.pl" className="text-primary underline">
              kontakt@openoms.pl
            </a>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
