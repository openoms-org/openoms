"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { useWorkflowTemplates } from "@/hooks/use-workflows";
import { useCreateAutomationRule } from "@/hooks/use-automation";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { ArrowLeft, Plus, Zap, FileText, Loader2 } from "lucide-react";
import type { WorkflowTemplate, WorkflowDefinition } from "@/types/api";
import { createEmptyWorkflow } from "@/lib/workflow-types";

export default function NewWorkflowPage() {
  const router = useRouter();
  const { data: templates, isLoading } = useWorkflowTemplates();
  const createRule = useCreateAutomationRule();
  const [creating, setCreating] = useState(false);

  const handleBlankWorkflow = () => {
    // Navigate to editor with no ID (create mode)
    // Store the empty definition in sessionStorage for the editor to pick up
    const def = createEmptyWorkflow();
    sessionStorage.setItem("workflow-new", JSON.stringify({
      name: "Nowy workflow",
      definition: def,
    }));
    router.push("/workflows/new-editor");
  };

  const handleTemplateSelect = async (template: WorkflowTemplate) => {
    // Store template definition in sessionStorage and navigate to editor
    sessionStorage.setItem("workflow-new", JSON.stringify({
      name: template.name,
      definition: template.definition,
    }));
    router.push("/workflows/new-editor");
  };

  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => router.push("/workflows")}>
            <ArrowLeft className="h-4 w-4" />
            Wróc
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Nowy workflow</h1>
            <p className="text-muted-foreground">
              Wybierz szablon lub zacznij od pustego canvas
            </p>
          </div>
        </div>

        {/* Blank canvas option */}
        <Card
          className="cursor-pointer hover:shadow-md transition-shadow border-dashed border-2"
          onClick={handleBlankWorkflow}
        >
          <CardContent className="flex items-center gap-4 py-6">
            <div className="p-3 rounded-lg bg-muted">
              <Plus className="h-8 w-8 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-lg font-semibold">Pusty canvas</h3>
              <p className="text-sm text-muted-foreground">
                Zacznij od zera - przeciagaj elementy na canvas
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Templates */}
        <div>
          <h2 className="text-lg font-semibold mb-3">Szablony</h2>
          {isLoading ? (
            <LoadingSkeleton />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {templates?.map((template: WorkflowTemplate) => (
                <Card
                  key={template.id}
                  className="cursor-pointer hover:shadow-md transition-shadow"
                  onClick={() => handleTemplateSelect(template)}
                >
                  <CardHeader>
                    <div className="flex items-center gap-2">
                      <div className="p-1.5 rounded bg-primary">
                        <FileText className="h-4 w-4 text-primary-foreground" />
                      </div>
                      <CardTitle className="text-base">{template.name}</CardTitle>
                    </div>
                    <CardDescription>{template.description}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{template.definition.nodes.length} wezlow</span>
                      <span className="text-muted-foreground/40">|</span>
                      <span>{template.definition.edges.length} polaczen</span>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </AdminGuard>
  );
}
