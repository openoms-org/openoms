"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  Send,
  MessageSquare,
  User,
  Mail,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroThreads,
  useAllegroMessages,
  useSendAllegroMessage,
} from "@/hooks/use-allegro";
import type { AllegroThread } from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn, formatDate } from "@/lib/utils";

export default function AllegroMessagesPage() {
  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null);

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
            <h1 className="text-2xl font-bold">Wiadomosci Allegro</h1>
            <p className="text-muted-foreground">
              Odpowiadaj na wiadomosci od kupujacych
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3 h-[calc(100vh-200px)]">
          <div className="lg:col-span-1 border rounded-lg overflow-hidden flex flex-col">
            <ThreadList
              selectedThreadId={selectedThreadId}
              onSelectThread={setSelectedThreadId}
            />
          </div>
          <div className="lg:col-span-2 border rounded-lg overflow-hidden flex flex-col">
            {selectedThreadId ? (
              <MessageView threadId={selectedThreadId} />
            ) : (
              <div className="flex-1 flex items-center justify-center text-muted-foreground">
                <div className="text-center space-y-2">
                  <MessageSquare className="h-12 w-12 mx-auto opacity-30" />
                  <p className="text-sm">Wybierz watek, aby wyswietlic wiadomosci</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </AdminGuard>
  );
}

function ThreadList({
  selectedThreadId,
  onSelectThread,
}: {
  selectedThreadId: string | null;
  onSelectThread: (id: string) => void;
}) {
  const { data, isLoading, isError, refetch } = useAllegroThreads({
    limit: 50,
  });

  if (isLoading) {
    return (
      <div className="p-4 space-y-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-4 text-center space-y-2">
        <p className="text-sm text-destructive">Blad ladowania watkow</p>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          Sprobuj ponownie
        </Button>
      </div>
    );
  }

  const threads = data?.threads ?? [];

  if (threads.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-4">
        <div className="text-center space-y-2">
          <Mail className="h-8 w-8 mx-auto text-muted-foreground opacity-30" />
          <p className="text-sm text-muted-foreground">Brak watkow</p>
        </div>
      </div>
    );
  }

  return (
    <ScrollArea className="flex-1">
      <div className="divide-y">
        {threads.map((thread: AllegroThread) => (
          <button
            key={thread.id}
            onClick={() => onSelectThread(thread.id)}
            className={cn(
              "w-full text-left px-4 py-3 hover:bg-accent transition-colors",
              selectedThreadId === thread.id && "bg-accent",
              !thread.read && "bg-primary/5"
            )}
          >
            <div className="flex items-start gap-3">
              <Avatar size="sm">
                <AvatarFallback>
                  {thread.interlocutor.login.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span
                    className={cn(
                      "text-sm truncate",
                      !thread.read && "font-semibold"
                    )}
                  >
                    {thread.interlocutor.login}
                  </span>
                  {!thread.read && (
                    <Badge
                      variant="default"
                      className="shrink-0 text-[10px] px-1.5 py-0"
                    >
                      Nowa
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground truncate mt-0.5">
                  {thread.subject || (thread.offer ? thread.offer.name : "Brak tematu")}
                </p>
                <p className="text-[11px] text-muted-foreground mt-0.5">
                  {formatDate(thread.lastMessageDateTime)}
                </p>
              </div>
            </div>
          </button>
        ))}
      </div>
    </ScrollArea>
  );
}

function MessageView({ threadId }: { threadId: string }) {
  const { data, isLoading, isError, refetch } = useAllegroMessages(threadId);
  const sendMessage = useSendAllegroMessage(threadId);
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

    sendMessage.mutate(text, {
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
    });
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center space-y-2">
          <p className="text-sm text-destructive">Blad ladowania wiadomosci</p>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            Sprobuj ponownie
          </Button>
        </div>
      </div>
    );
  }

  const messages = [...(data?.messages ?? [])].reverse();

  return (
    <>
      {/* Messages area */}
      <ScrollArea className="flex-1 p-4">
        <div className="space-y-3">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={cn(
                "flex",
                msg.author.isInterlocutor ? "justify-start" : "justify-end"
              )}
            >
              <div
                className={cn(
                  "max-w-[75%] rounded-lg px-3 py-2",
                  msg.author.isInterlocutor
                    ? "bg-muted text-foreground"
                    : "bg-primary text-primary-foreground"
                )}
              >
                <div className="flex items-center gap-2 mb-1">
                  {msg.author.isInterlocutor && (
                    <User className="h-3 w-3 shrink-0" />
                  )}
                  <span className="text-[11px] font-medium">
                    {msg.author.login}
                  </span>
                </div>
                <p className="text-sm whitespace-pre-wrap break-words">
                  {msg.text}
                </p>
                <p
                  className={cn(
                    "text-[10px] mt-1",
                    msg.author.isInterlocutor
                      ? "text-muted-foreground"
                      : "text-primary-foreground/70"
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

      {/* Reply form */}
      <div className="border-t p-3">
        <div className="flex gap-2">
          <Textarea
            value={messageText}
            onChange={(e) => setMessageText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Napisz wiadomosc... (Enter aby wyslac, Shift+Enter nowa linia)"
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
    </>
  );
}
