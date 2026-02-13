"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  ChevronRight,
  Folder,
  FolderOpen,
  Loader2,
  Search,
  Tag,
  Image as ImageIcon,
  Info,
} from "lucide-react";
import { AdminGuard } from "@/components/shared/admin-guard";
import {
  useAllegroCategories,
  useAllegroCategoryParams,
  useAllegroProductSearch,
} from "@/hooks/use-allegro";
import type {
  AllegroCategory,
  AllegroCategoryParameter,
} from "@/hooks/use-allegro";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

interface BreadcrumbItem {
  id: string;
  name: string;
}

export default function AllegroCatalogPage() {
  const [parentId, setParentId] = useState<string | null>(null);
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbItem[]>([]);
  const [selectedLeafId, setSelectedLeafId] = useState<string | null>(null);
  const [searchPhrase, setSearchPhrase] = useState("");
  const [activeSearch, setActiveSearch] = useState("");
  const [searchCategoryId, setSearchCategoryId] = useState<string | null>(null);

  const { data: categoriesData, isLoading: categoriesLoading } =
    useAllegroCategories(parentId);

  const { data: paramsData, isLoading: paramsLoading } =
    useAllegroCategoryParams(selectedLeafId);

  const { data: productsData, isLoading: productsLoading } =
    useAllegroProductSearch(
      activeSearch
        ? {
            phrase: activeSearch,
            category_id: searchCategoryId ?? undefined,
            limit: 20,
          }
        : undefined
    );

  const handleCategoryClick = (cat: AllegroCategory) => {
    if (cat.leaf) {
      setSelectedLeafId(selectedLeafId === cat.id ? null : cat.id);
      setSearchCategoryId(cat.id);
    } else {
      setParentId(cat.id);
      setBreadcrumb((prev) => [...prev, { id: cat.id, name: cat.name }]);
      setSelectedLeafId(null);
    }
  };

  const handleBreadcrumbClick = (index: number) => {
    if (index < 0) {
      setParentId(null);
      setBreadcrumb([]);
    } else {
      const item = breadcrumb[index];
      setParentId(item.id);
      setBreadcrumb((prev) => prev.slice(0, index + 1));
    }
    setSelectedLeafId(null);
  };

  const handleSearch = () => {
    setActiveSearch(searchPhrase.trim());
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/integrations/allegro">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Katalog Allegro</h1>
            <p className="text-muted-foreground">
              Przegladaj kategorie i wyszukuj produkty w katalogu Allegro
            </p>
          </div>
        </div>

        {/* Help section */}
        <div className="rounded-lg border bg-muted/50 p-4 flex gap-3">
          <Info className="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div className="space-y-1 text-sm">
            <p className="font-medium">Jak korzystac z katalogu?</p>
            <ul className="list-disc list-inside space-y-0.5 text-muted-foreground">
              <li>Przegladaj drzewo kategorii Allegro klikajac w foldery. Kategorie oznaczone &quot;Lisc&quot; to kategorie koncowe.</li>
              <li>Po wybraniu kategorii koncowej zobaczysz wymagane parametry (np. rozmiar, kolor, material) — to pola, ktore musisz wypelnic w ofercie.</li>
              <li>Wyszukiwarka produktow pozwala znalezc produkty w katalogu Allegro po nazwie. Jesli wybierzesz kategorie, wyszukiwanie zawezi sie do niej.</li>
              <li>Katalog sluzy do podgladu — tworzenie i edycja ofert odbywa sie na stronie &quot;Oferty&quot;.</li>
            </ul>
          </div>
        </div>

        {/* Category Browser */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Folder className="h-5 w-5" />
              Kategorie
            </CardTitle>
          </CardHeader>
          <CardContent>
            {/* Breadcrumb */}
            <div className="mb-4 flex flex-wrap items-center gap-1 text-sm">
              <button
                onClick={() => handleBreadcrumbClick(-1)}
                className="text-primary hover:underline font-medium"
              >
                Wszystkie kategorie
              </button>
              {breadcrumb.map((item, idx) => (
                <span key={item.id} className="flex items-center gap-1">
                  <ChevronRight className="h-3 w-3 text-muted-foreground" />
                  <button
                    onClick={() => handleBreadcrumbClick(idx)}
                    className={
                      idx === breadcrumb.length - 1
                        ? "font-medium"
                        : "text-primary hover:underline"
                    }
                  >
                    {item.name}
                  </button>
                </span>
              ))}
            </div>

            {/* Category List */}
            {categoriesLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            ) : !categoriesData?.categories?.length ? (
              <p className="py-4 text-center text-muted-foreground">
                Brak kategorii do wyswietlenia
              </p>
            ) : (
              <div className="grid grid-cols-1 gap-1 sm:grid-cols-2 lg:grid-cols-3">
                {categoriesData.categories.map((cat) => (
                  <button
                    key={cat.id}
                    onClick={() => handleCategoryClick(cat)}
                    className={`flex items-center gap-3 rounded-md border p-3 text-left text-sm transition-colors hover:bg-muted/50 ${
                      selectedLeafId === cat.id
                        ? "border-primary bg-primary/5"
                        : "border-transparent"
                    }`}
                  >
                    {cat.leaf ? (
                      <Tag className="h-4 w-4 shrink-0 text-muted-foreground" />
                    ) : (
                      <FolderOpen className="h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                    <span className="flex-1 truncate">{cat.name}</span>
                    {cat.leaf ? (
                      <Badge variant="secondary" className="text-xs shrink-0">
                        Lisc
                      </Badge>
                    ) : (
                      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                  </button>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Category Parameters */}
        {selectedLeafId && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Tag className="h-5 w-5" />
                Parametry wymagane
                {paramsLoading && (
                  <Loader2 className="ml-2 h-4 w-4 animate-spin" />
                )}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {paramsLoading ? (
                <div className="space-y-2">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-8 w-full" />
                  ))}
                </div>
              ) : !paramsData?.parameters?.length ? (
                <p className="py-4 text-center text-muted-foreground">
                  Brak parametrow dla tej kategorii
                </p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Nazwa</TableHead>
                      <TableHead className="w-28">Typ</TableHead>
                      <TableHead className="w-28">Wymagany</TableHead>
                      <TableHead className="w-24">Jednostka</TableHead>
                      <TableHead className="w-32">Slownik</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {paramsData.parameters.map((param) => (
                      <ParameterRow key={param.id} param={param} />
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        )}

        {/* Product Search */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Search className="h-5 w-5" />
              Szukaj produktow w katalogu
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-3 sm:flex-row">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Wpisz nazwe produktu..."
                  value={searchPhrase}
                  onChange={(e) => setSearchPhrase(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                  className="pl-10"
                />
              </div>
              <Button onClick={handleSearch} disabled={!searchPhrase.trim()}>
                Szukaj
              </Button>
            </div>

            {searchCategoryId && (
              <p className="text-xs text-muted-foreground">
                Wyszukiwanie w wybranej kategorii (ID: {searchCategoryId})
              </p>
            )}

            {productsLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            ) : activeSearch && !productsData?.products?.length ? (
              <p className="py-8 text-center text-muted-foreground">
                Nie znaleziono produktow dla &quot;{activeSearch}&quot;
              </p>
            ) : productsData?.products?.length ? (
              <>
                <p className="text-sm text-muted-foreground">
                  Znaleziono: {productsData.count} produktow
                </p>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-16">Zdjecie</TableHead>
                      <TableHead>Nazwa</TableHead>
                      <TableHead className="w-32">Kategoria</TableHead>
                      <TableHead className="w-24">ID</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {productsData.products.map((product) => (
                      <TableRow key={product.id}>
                        <TableCell>
                          {product.images?.[0]?.url ? (
                            <img
                              src={product.images[0].url}
                              alt={product.name}
                              className="h-12 w-12 rounded object-cover"
                            />
                          ) : (
                            <div className="flex h-12 w-12 items-center justify-center rounded bg-muted">
                              <ImageIcon className="h-5 w-5 text-muted-foreground" />
                            </div>
                          )}
                        </TableCell>
                        <TableCell>
                          <p className="font-medium text-sm line-clamp-2">
                            {product.name}
                          </p>
                        </TableCell>
                        <TableCell>
                          {product.category?.id ? (
                            <Badge variant="outline" className="text-xs">
                              {product.category.id}
                            </Badge>
                          ) : (
                            "---"
                          )}
                        </TableCell>
                        <TableCell>
                          <code className="text-xs">{product.id}</code>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </AdminGuard>
  );
}

function ParameterRow({ param }: { param: AllegroCategoryParameter }) {
  const [showDict, setShowDict] = useState(false);
  const hasDictionary = param.dictionary && param.dictionary.length > 0;

  return (
    <>
      <TableRow>
        <TableCell className="font-medium">{param.name}</TableCell>
        <TableCell>
          <Badge variant="outline" className="text-xs">
            {param.type}
          </Badge>
        </TableCell>
        <TableCell>
          {param.required ? (
            <Badge variant="default" className="text-xs">
              Tak
            </Badge>
          ) : (
            <span className="text-muted-foreground text-xs">Nie</span>
          )}
        </TableCell>
        <TableCell>
          <span className="text-xs">{param.unit || "---"}</span>
        </TableCell>
        <TableCell>
          {hasDictionary ? (
            <button
              onClick={() => setShowDict(!showDict)}
              className="text-xs text-primary hover:underline"
            >
              {showDict ? "Ukryj" : `Pokaz (${param.dictionary!.length})`}
            </button>
          ) : (
            <span className="text-xs text-muted-foreground">---</span>
          )}
        </TableCell>
      </TableRow>
      {showDict && hasDictionary && (
        <TableRow>
          <TableCell colSpan={5}>
            <div className="flex flex-wrap gap-1 py-1">
              {param.dictionary!.map((d) => (
                <Badge key={d.id} variant="secondary" className="text-xs">
                  {d.value}
                </Badge>
              ))}
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}
