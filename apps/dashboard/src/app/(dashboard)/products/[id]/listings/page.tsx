import { Card, CardContent } from "@/components/ui/card";

export default function ProductListingsPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Oferty marketplace</h1>
      <Card>
        <CardContent className="py-8 text-center text-muted-foreground">
          <p>Zarządzanie ofertami marketplace będzie dostępne wkrótce.</p>
          <p className="text-sm mt-2">Tutaj będziesz mógł powiązać produkt z ofertami na Allegro, Amazon, eBay i innych platformach.</p>
        </CardContent>
      </Card>
    </div>
  );
}
