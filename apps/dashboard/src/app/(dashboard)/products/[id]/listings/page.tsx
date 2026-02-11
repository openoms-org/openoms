"use client";

import { useParams } from "next/navigation";
import { redirect } from "next/navigation";

export default function ProductListingsPage() {
  const params = useParams<{ id: string }>();
  redirect(`/products/${params.id}`);
}
