"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { pl } from "date-fns/locale";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/shared/status-badge";
import { ORDER_STATUSES } from "@/lib/constants";
import { useOrderStatuses, statusesToMap } from "@/hooks/use-order-statuses";
import type { OrderSummary } from "@/types/api";

interface RecentOrdersTableProps {
  orders?: OrderSummary[];
  isLoading: boolean;
}

export function RecentOrdersTable({ orders, isLoading }: RecentOrdersTableProps) {
  const { data: statusConfig } = useOrderStatuses();
  const orderStatuses = statusConfig ? statusesToMap(statusConfig) : ORDER_STATUSES;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ostatnie zamówienia</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : !orders || orders.length === 0 ? (
          <div className="flex h-32 items-center justify-center text-muted-foreground">
            Brak zamówień
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Klient</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Źródło</TableHead>
                <TableHead>Kwota</TableHead>
                <TableHead>Data</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders.map((order) => (
                <TableRow key={order.id}>
                  <TableCell>
                    <Link
                      href={`/orders/${order.id}`}
                      className="font-medium text-primary hover:underline"
                    >
                      {order.customer_name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={order.status} statusMap={orderStatuses} />
                  </TableCell>
                  <TableCell className="capitalize">{order.source}</TableCell>
                  <TableCell>
                    {order.total_amount.toFixed(2)} {order.currency}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDistanceToNow(new Date(order.created_at), {
                      addSuffix: true,
                      locale: pl,
                    })}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
