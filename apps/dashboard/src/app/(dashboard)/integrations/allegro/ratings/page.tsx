"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Loader2,
  RotateCcw,
  Star,
  CheckCircle2,
  XCircle,
  Minus,
  MessageSquare,
  Trash2,
  Flag,
  ChevronDown,
  ChevronUp,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroRatings,
  useAllegroRatingAnswer,
  useCreateAllegroRatingAnswer,
  useDeleteAllegroRatingAnswer,
  useRequestAllegroRatingRemoval,
} from "@/hooks/use-allegro";
import type { AllegroUserRating } from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { EmptyState } from "@/components/shared/empty-state";
import { formatDate } from "@/lib/utils";

function RateIcon({ rate }: { rate: string }) {
  switch (rate) {
    case "POSITIVE":
      return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case "NEGATIVE":
      return <XCircle className="h-4 w-4 text-red-500" />;
    case "NEUTRAL":
      return <Minus className="h-4 w-4 text-gray-400" />;
    default:
      return <Minus className="h-4 w-4 text-gray-400" />;
  }
}

function rateLabel(rate: string): string {
  switch (rate) {
    case "POSITIVE":
      return "Pozytywna";
    case "NEGATIVE":
      return "Negatywna";
    case "NEUTRAL":
      return "Neutralna";
    default:
      return rate;
  }
}

export default function AllegroRatingsPage() {
  const [limit] = useState(25);
  const [offset, setOffset] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [answerDialogRating, setAnswerDialogRating] =
    useState<AllegroUserRating | null>(null);
  const [deleteDialogRatingId, setDeleteDialogRatingId] = useState<
    string | null
  >(null);
  const [removalDialogRatingId, setRemovalDialogRatingId] = useState<
    string | null
  >(null);

  const { data, isLoading, isError, refetch } = useAllegroRatings({
    limit,
    offset,
  });

  const ratings = data?.ratings ?? [];

  // Compute summary stats
  const total = ratings.length;
  const positive = ratings.filter((r) => r.rate === "POSITIVE").length;
  const negative = ratings.filter((r) => r.rate === "NEGATIVE").length;
  const neutral = ratings.filter((r) => r.rate === "NEUTRAL").length;
  const pctPositive = total > 0 ? Math.round((positive / total) * 100) : 0;
  const pctNegative = total > 0 ? Math.round((negative / total) * 100) : 0;

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
            <h1 className="text-2xl font-bold">Oceny sprzedawcy</h1>
            <p className="text-muted-foreground">
              Zarzadzaj ocenami i odpowiedziami na Allegro
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Zarzadzanie ocenami</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li>Tu widzisz oceny wystawione przez kupujacych. Kliknij w ocene, aby rozwinac szczegoly i dostepne akcje.</li>
              <li><strong>Odpowiedz na ocene</strong> — napisz komentarz widoczny publicznie pod ocena. Mozesz edytowac lub usunac swoja odpowiedz.</li>
              <li><strong>Zglos do usuniecia</strong> — jesli ocena narusza regulamin Allegro (np. wulgaryzmy, nieprawdziwe informacje), mozesz ja zglosic do weryfikacji.</li>
              <li>Statystyki u gory pokazuja podsumowanie ocen z aktualnej strony. Dbaj o wysoki procent pozytywnych ocen — wplywa na widocznosc ofert.</li>
            </ul>
          </div>
        </div>

        {/* Summary stats */}
        {!isLoading && ratings.length > 0 && (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  Lacznie
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold">{total}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  Pozytywne
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-green-600">
                  {positive}{" "}
                  <span className="text-sm font-normal text-muted-foreground">
                    ({pctPositive}%)
                  </span>
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  Negatywne
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-red-600">
                  {negative}{" "}
                  <span className="text-sm font-normal text-muted-foreground">
                    ({pctNegative}%)
                  </span>
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  Neutralne
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-bold text-gray-500">{neutral}</p>
              </CardContent>
            </Card>
          </div>
        )}

        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RotateCcw className="mr-2 h-4 w-4" />
            Odswiez
          </Button>
        </div>

        {isError && (
          <Card className="border-destructive">
            <CardContent className="pt-4">
              <p className="text-sm text-destructive">
                Blad podczas ladowania ocen. Sprawdz polaczenie z Allegro.
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

        {!isLoading && ratings.length === 0 && (
          <EmptyState
            icon={Star}
            title="Brak ocen"
            description="Nie znaleziono ocen do wyswietlenia."
          />
        )}

        {!isLoading && ratings.length > 0 && (
          <div className="space-y-2">
            {ratings.map((rating) => (
              <RatingCard
                key={rating.id}
                rating={rating}
                isExpanded={expandedId === rating.id}
                onToggle={() =>
                  setExpandedId(expandedId === rating.id ? null : rating.id)
                }
                onAnswer={() => setAnswerDialogRating(rating)}
                onDeleteAnswer={() => setDeleteDialogRatingId(rating.id)}
                onRequestRemoval={() => setRemovalDialogRatingId(rating.id)}
              />
            ))}
          </div>
        )}

        {/* Pagination */}
        {data && data.count > limit && (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Wyswietlanie {offset + 1}-{Math.min(offset + limit, data.count)}{" "}
              z {data.count}
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

        {/* Answer dialog */}
        {answerDialogRating && (
          <AnswerDialog
            rating={answerDialogRating}
            onClose={() => setAnswerDialogRating(null)}
          />
        )}

        {/* Delete answer dialog */}
        {deleteDialogRatingId && (
          <DeleteAnswerDialog
            ratingId={deleteDialogRatingId}
            onClose={() => setDeleteDialogRatingId(null)}
          />
        )}

        {/* Removal request dialog */}
        {removalDialogRatingId && (
          <RemovalDialog
            ratingId={removalDialogRatingId}
            onClose={() => setRemovalDialogRatingId(null)}
          />
        )}
      </div>
    </AdminGuard>
  );
}

function RatingCard({
  rating,
  isExpanded,
  onToggle,
  onAnswer,
  onDeleteAnswer,
  onRequestRemoval,
}: {
  rating: AllegroUserRating;
  isExpanded: boolean;
  onToggle: () => void;
  onAnswer: () => void;
  onDeleteAnswer: () => void;
  onRequestRemoval: () => void;
}) {
  return (
    <Card>
      <CardContent className="pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <RateIcon rate={rating.rate} />
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm">
                  {rating.buyer.login}
                </span>
                <Badge
                  variant={
                    rating.rate === "POSITIVE"
                      ? "default"
                      : rating.rate === "NEGATIVE"
                        ? "destructive"
                        : "secondary"
                  }
                >
                  {rateLabel(rating.rate)}
                </Badge>
              </div>
              <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                <span>
                  Zamowienie: {rating.order.id.slice(0, 12)}
                </span>
                <span>{formatDate(rating.createdAt)}</span>
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
            <div className="space-y-3">
              {/* Full comment */}
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">
                  Komentarz
                </p>
                <p className="text-sm">
                  {rating.comment || (
                    <span className="text-muted-foreground italic">
                      Brak komentarza
                    </span>
                  )}
                </p>
              </div>

              {/* Existing answer */}
              <RatingAnswerSection ratingId={rating.id} />

              {/* Actions */}
              <div className="flex gap-2 flex-wrap">
                <Button variant="outline" size="sm" onClick={onAnswer}>
                  <MessageSquare className="mr-2 h-4 w-4" />
                  Odpowiedz na ocene
                </Button>
                <Button variant="outline" size="sm" onClick={onDeleteAnswer}>
                  <Trash2 className="mr-2 h-4 w-4" />
                  Usun odpowiedz
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onRequestRemoval}
                >
                  <Flag className="mr-2 h-4 w-4" />
                  Zglos do usuniecia
                </Button>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function RatingAnswerSection({ ratingId }: { ratingId: string }) {
  const { data, isLoading } = useAllegroRatingAnswer(ratingId);

  if (isLoading) {
    return <Skeleton className="h-12 w-full" />;
  }

  if (!data || !data.text) {
    return null;
  }

  return (
    <div className="rounded-md border p-3 bg-muted/50">
      <p className="text-xs font-medium text-muted-foreground mb-1">
        Twoja odpowiedz
      </p>
      <p className="text-sm">{data.text}</p>
      {data.createdAt && (
        <p className="text-[10px] text-muted-foreground mt-1">
          {formatDate(data.createdAt)}
        </p>
      )}
    </div>
  );
}

function AnswerDialog({
  rating,
  onClose,
}: {
  rating: AllegroUserRating;
  onClose: () => void;
}) {
  const [text, setText] = useState("");
  const createAnswer = useCreateAllegroRatingAnswer();

  const handleSubmit = () => {
    if (!text.trim()) {
      toast.error("Podaj tresc odpowiedzi");
      return;
    }

    createAnswer.mutate(
      { ratingId: rating.id, text: text.trim() },
      {
        onSuccess: () => {
          toast.success("Odpowiedz zostala zapisana");
          onClose();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie zapisac odpowiedzi"
          );
        },
      }
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Odpowiedz na ocene</DialogTitle>
          <DialogDescription>
            Ocena od {rating.buyer.login} ({rateLabel(rating.rate)})
          </DialogDescription>
        </DialogHeader>

        {rating.comment && (
          <div className="rounded-md border p-3 bg-muted/50">
            <p className="text-xs font-medium text-muted-foreground mb-1">
              Komentarz kupujacego
            </p>
            <p className="text-sm">{rating.comment}</p>
          </div>
        )}

        <div className="space-y-3">
          <div>
            <Label htmlFor="answer-text">Twoja odpowiedz</Label>
            <Textarea
              id="answer-text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Napisz odpowiedz na ocene..."
              className="mt-1"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!text.trim() || createAnswer.isPending}
          >
            {createAnswer.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Wyslij
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DeleteAnswerDialog({
  ratingId,
  onClose,
}: {
  ratingId: string;
  onClose: () => void;
}) {
  const deleteAnswer = useDeleteAllegroRatingAnswer();

  const handleDelete = () => {
    deleteAnswer.mutate(ratingId, {
      onSuccess: () => {
        toast.success("Odpowiedz zostala usunieta");
        onClose();
      },
      onError: (error) => {
        toast.error(
          error instanceof Error
            ? error.message
            : "Nie udalo sie usunac odpowiedzi"
        );
      },
    });
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Usun odpowiedz</DialogTitle>
          <DialogDescription>
            Czy na pewno chcesz usunac swoja odpowiedz na te ocene? Tej operacji
            nie mozna cofnac.
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteAnswer.isPending}
          >
            {deleteAnswer.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Usun odpowiedz
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RemovalDialog({
  ratingId,
  onClose,
}: {
  ratingId: string;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const requestRemoval = useRequestAllegroRatingRemoval();

  const handleSubmit = () => {
    if (!reason.trim()) {
      toast.error("Podaj powod zgloszenia");
      return;
    }

    requestRemoval.mutate(
      { ratingId, reason: reason.trim() },
      {
        onSuccess: () => {
          toast.success("Zgloszenie zostalo wyslane");
          onClose();
        },
        onError: (error) => {
          toast.error(
            error instanceof Error
              ? error.message
              : "Nie udalo sie wyslac zgloszenia"
          );
        },
      }
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Zglos do usuniecia</DialogTitle>
          <DialogDescription>
            Zglos te ocene do weryfikacji i ewentualnego usuniecia przez Allegro.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div>
            <Label htmlFor="removal-reason">Powod zgloszenia</Label>
            <Textarea
              id="removal-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Opisz powod zgloszenia oceny do usuniecia..."
              className="mt-1"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Anuluj
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!reason.trim() || requestRemoval.isPending}
          >
            {requestRemoval.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Zglos do usuniecia
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
