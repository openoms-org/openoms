"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ArrowUpRight,
  BookOpen,
  CheckCircle2,
  Code2,
  FileQuestion,
  LifeBuoy,
  Mail,
} from "lucide-react";
import Link from "next/link";
import { useTranslations } from "next-intl";

const SUPPORT_EMAIL = "support@openoms.org";
const USER_GUIDE_URL =
  "https://github.com/openoms-org/openoms/blob/main/docs/poradnik-uzytkownika.md";
const FAQ_URL = "https://github.com/openoms-org/openoms/blob/main/docs/faq-sprzedawcy.md";
const GITHUB_ISSUES_URL = "https://github.com/openoms-org/openoms/issues";

export default function HelpPage() {
  const t = useTranslations("help");

  const resourceLinks = [
    {
      titleKey: "userGuide",
      descriptionKey: "userGuideDescription",
      href: USER_GUIDE_URL,
      icon: BookOpen,
    },
    {
      titleKey: "faq",
      descriptionKey: "faqDescription",
      href: FAQ_URL,
      icon: FileQuestion,
    },
  ];

  const preparationItems = ["prepareTenant", "prepareContext", "prepareEvidence"];

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="mt-1 max-w-2xl text-muted-foreground">{t("subtitle")}</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1.5fr_1fr]">
        <Card className="border-info/20 bg-info/5">
          <CardHeader className="gap-3">
            <div className="flex size-11 items-center justify-center rounded-lg bg-info text-info-foreground">
              <LifeBuoy className="size-5" aria-hidden="true" />
            </div>
            <div className="space-y-1">
              <CardTitle className="text-xl">{t("officialSupportTitle")}</CardTitle>
              <p className="text-sm leading-6 text-muted-foreground">
                {t("officialSupportDescription")}
              </p>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <Button asChild size="lg" className="w-full sm:w-auto">
              <a href={`mailto:${SUPPORT_EMAIL}`}>
                <Mail className="size-4" aria-hidden="true" />
                {t("officialSupportCta")}
              </a>
            </Button>
            <p className="text-sm text-muted-foreground">{t("officialSupportMeta")}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("prepareTitle")}</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-3">
              {preparationItems.map((item) => (
                <li key={item} className="flex gap-3 text-sm text-muted-foreground">
                  <CheckCircle2
                    className="mt-0.5 size-4 shrink-0 text-info"
                    aria-hidden="true"
                  />
                  <span>{t(item)}</span>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {resourceLinks.map((link) => (
          <Link
            key={link.titleKey}
            href={link.href}
            target="_blank"
            rel="noopener noreferrer"
          >
            <Card className="h-full cursor-pointer transition-colors hover:bg-accent/50">
              <CardHeader className="flex flex-row items-center gap-3 pb-2">
                <link.icon className="size-5 text-info" aria-hidden="true" />
                <CardTitle className="text-base">{t(link.titleKey)}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm leading-6 text-muted-foreground">
                  {t(link.descriptionKey)}
                </p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <section>
        <Card className="bg-muted/30">
          <CardHeader className="gap-2">
            <div className="flex items-center gap-2">
              <Code2 className="size-5 text-muted-foreground" aria-hidden="true" />
              <CardTitle className="text-base">{t("ossTitle")}</CardTitle>
            </div>
            <p className="text-sm leading-6 text-muted-foreground">{t("ossDescription")}</p>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline">
              <Link href={GITHUB_ISSUES_URL} target="_blank" rel="noopener noreferrer">
                {t("reportBug")}
                <ArrowUpRight className="size-4" aria-hidden="true" />
              </Link>
            </Button>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
