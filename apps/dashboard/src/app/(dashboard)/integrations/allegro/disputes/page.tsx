"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Send,
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  RotateCcw,
  MessageSquare,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroDisputes,
  useAllegroDisputeMessages,
  useSendAllegroDisputeMessage,
} from "@/hooks/use-allegro";
import type {
  AllegroDispute,
  AllegroDisputeMessage,
} from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import { EmptyState } from "@/components/shared/empty-state";
import { cn, formatDate } from "@/lib/utils";

const DISPUTE_STATUS_MAP: Record<
  string,
  { label: string; variant: "default" | "secondary" | "destructive" | "outline" }
> = {
  OPEN: { label: "Otwarty", variant: "destructive" },
  CLOSED: { label: "Zamkniety", variant: "secondary" },
};

export default function AllegroDisputesPage() {
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [limit] = useState(25);
  const [offset, setOffset] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data, isLoading, isError, refetch } = useAllegroDisputes({
    limit,
    offset,
    status: statusFilter || undefined,
  });

  const disputes = data?.disputes ?? [];

  return (
    <AdminGuard>
      <div className="space-y-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Spory i reklamacje</h1>
            <p className="text-muted-foreground">
              Zarzadzaj sporami z kupujacymi na Allegro
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Spory z kupujacymi</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li>Spory (dyskusje) sa otwierane przez kupujacych, gdy maja problem z transakcja (np. towar nie dotarl, niezgodnosc z opisem).</li>
              <li>Kliknij w spor, aby rozwinac historie wiadomosci i odpowiedziec kupujacemu.</li>
              <li>Odpowiadaj szybko — Allegro ocenia czas reakcji sprzedawcy. Brak odpowiedzi moze skutkowac niekorzystnym rozstrzygnieciem.</li>
              <li>Filtruj po statusie: <strong>Otwarte</strong> (wymagaja Twojej reakcji) lub <strong>Zamkniete</strong> (zakonczone).</li>
            </ul>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <div className="w-[200px]">
            <Select
              value={statusFilter || "all"}
              onValueChange={(v) => {
                setStatusFilter(v === "all" ? "" : v);
                setOffset(0);
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Wszystkie</SelectItem>
                <SelectItem value="OPEN">Otwarte</SelectItem>
                <SelectItem value="CLOSED">Zamkniete</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            Odswiez
          </Button>
        </div>

        {isError && (
          <Card className="border-destructive">
            <CardContent className="pt-4">
              <p className="text-sm text-destructive">
                Blad podczas ladowania sporow. Sprawdz polaczenie z Allegro.
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-2"
                onClick={() => refetch()}
              >
                Sprobuj ponownie
              </Button>
            </CardContent>
          </Card>
        )}

        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full" />
            ))}
          </div>
        )}

        {!isLoading && disputes.length === 0 && (
          <EmptyState
            icon={AlertTriangle}
            title="Brak sporow"
            description="Nie znaleziono sporow do wyswietlenia."
          />
        )}

        {!isLoading && disputes.length > 0 && (
          <div className="space-y-2">
            {disputes.map((dispute) => (
              <DisputeCard
                key={dispute.id}
                dispute={dispute}
                isExpanded={expandedId === dispute.id}
                onToggle={() =>
                  setExpandedId(expandedId === dispute.id ? null : dispute.id)
                }
              />
            ))}
          </div>
        )}

        {/* Pagination */}
        {data && data.count > limit && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Wyswietlanie {offset + 1}-{Math.min(offset + limit, data.count)} z{" "}
              {data.count}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
              >
                Poprzednia
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={offset + limit >= data.count}
                onClick={() => setOffset(offset + limit)}
              >
                Nastepna
              </Button>
            </div>
          </div>
        )}
      </div>
    </AdminGuard>
  );
}

function DisputeCard({
  dispute,
  isExpanded,
  onToggle,
}: {
  dispute: AllegroDispute;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const statusInfo = DISPUTE_STATUS_MAP[dispute.status] ?? {
    label: dispute.status,
    variant: "outline" as const,
  };

  return (
    <Card>
      <CardContent className="pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm">
                  {dispute.subject || `Spor #${dispute.id.slice(0, 8)}`}
                </span>
                <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
              </div>
              <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                <span>Kupujacy: {dispute.buyer.login}</span>
                <span>Zamowienie: {dispute.checkoutForm.id.slice(0, 12)}</span>
                <span>{formatDate(dispute.createdAt)}</span>
              </div>
            </div>
          </div>
          <Button variant="ghost" size="icon" onClick={onToggle}>
            {isExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </Button>
        </div>

        {isExpanded && (
          <>
            <Separator className="my-3" />
            <DisputeMessages disputeId={dispute.id} />
          </>
        )}
      </CardContent>
    </Card>
  );
}

function DisputeMessages({ disputeId }: { disputeId: string }) {
  const { data, isLoading, isError, refetch } =
    useAllegroDisputeMessages(disputeId);
  const sendMessage = useSendAllegroDisputeMessage(disputeId);
  const [messageText, setMessageText] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (data?.messages) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [data?.messages]);

  const handleSend = () => {
    const text = messageText.trim();
    if (!text) return;

    sendMessage.mutate(
      { text },
      {
        onSuccess: () => {
          setMessageText("");
          toast.success("Wiadomosc wyslana");
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie wyslac wiadomosci"
          );
        },
      }
    );
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="text-center py-4 space-y-2">
        <p className="text-sm text-destructive">
          Blad ladowania wiadomosci sporu
        </p>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          Sprobuj ponownie
        </Button>
      </div>
    );
  }

  const messages = data?.messages ?? [];

  return (
    <div className="space-y-3">
      {messages.length === 0 ? (
        <div className="flex items-center justify-center py-4 text-muted-foreground">
          <div className="text-center space-y-1">
            <MessageSquare className="h-8 w-8 mx-auto opacity-30" />
            <p className="text-sm">Brak wiadomosci w tym sporze</p>
          </div>
        </div>
      ) : (
        <ScrollArea className="max-h-[400px]">
          <div className="space-y-2 p-1">
            {messages.map((msg: AllegroDisputeMessage) => (
              <div
                key={msg.id}
                className={cn(
                  "flex",
                  msg.author === "SELLER" ? "justify-end" : "justify-start"
                )}
              >
                <div
                  className={cn(
                    "max-w-[75%] rounded-lg px-3 py-2",
                    msg.author === "SELLER"
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted text-foreground"
                  )}
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-[11px] font-medium">
                      {msg.author === "SELLER" ? "Sprzedawca" : "Kupujacy"}
                    </span>
                    {msg.type && msg.type !== "MESSAGE" && (
                      <Badge
                        variant="outline"
                        className="text-[9px] px-1 py-0"
                      >
                        {msg.type}
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm whitespace-pre-wrap break-words">
                    {msg.text}
                  </p>
                  <p
                    className={cn(
                      "text-[10px] mt-1",
                      msg.author === "SELLER"
                        ? "text-primary-foreground/70"
                        : "text-muted-foreground"
                    )}
                  >
                    {formatDate(msg.createdAt)}
                  </p>
                </div>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>
      )}

      {/* Reply form */}
      <div className="border-t pt-3">
        <div className="flex gap-2">
          <Textarea
            value={messageText}
            onChange={(e) => setMessageText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Odpowiedz... (Enter aby wyslac, Shift+Enter nowa linia)"
            className="min-h-[60px] max-h-[120px] resize-none"
            disabled={sendMessage.isPending}
          />
          <Button
            onClick={handleSend}
            disabled={!messageText.trim() || sendMessage.isPending}
            size="icon"
            className="shrink-0 self-end"
          >
            {sendMessage.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
