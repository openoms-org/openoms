"use client";

import { useOnboarding } from "@/hooks/use-onboarding";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { CheckCircle2, Circle, X, ArrowRight } from "lucide-react";

export function OnboardingWizard() {
  const { steps, completedCount, isVisible, dismiss } = useOnboarding();

  if (!isVisible) return null;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div>
          <CardTitle className="text-lg">Pierwsze kroki</CardTitle>
          <p className="text-sm text-muted-foreground mt-1">
            {completedCount} z {steps.length} uko\u0144czone
          </p>
        </div>
        <Button variant="ghost" size="icon" onClick={() => dismiss()} title="Pomi\u0144">
          <X className="h-4 w-4" />
        </Button>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {steps.map((step) => (
            <div key={step.key} className="flex items-start gap-3">
              {step.completed ? (
                <CheckCircle2 className="h-5 w-5 text-green-500 mt-0.5 shrink-0" />
              ) : (
                <Circle className="h-5 w-5 text-muted-foreground mt-0.5 shrink-0" />
              )}
              <div className="flex-1 min-w-0">
                <p className={`text-sm font-medium ${step.completed ? "line-through text-muted-foreground" : ""}`}>
                  {step.title}
                </p>
                <p className="text-xs text-muted-foreground">{step.description}</p>
              </div>
              {!step.completed && (
                <Button variant="outline" size="sm" asChild>
                  <Link href={step.href}>
                    <ArrowRight className="h-3 w-3" />
                  </Link>
                </Button>
              )}
            </div>
          ))}
        </div>
        <div className="mt-4 flex h-2 rounded-full bg-muted overflow-hidden">
          <div
            className="bg-primary transition-all duration-300"
            style={{ width: `${(completedCount / steps.length) * 100}%` }}
          />
        </div>
      </CardContent>
    </Card>
  );
}
