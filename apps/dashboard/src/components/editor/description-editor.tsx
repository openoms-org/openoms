"use client";

import "./editor.css";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { Button } from "@/components/ui/button";
import {
  Heading1,
  Heading2,
  Pilcrow,
  List,
  ListOrdered,
  Sparkles,
  Loader2,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";

interface DescriptionEditorProps {
  value: string;
  onChange: (html: string) => void;
  placeholder?: string;
  onAiGenerate?: () => void;
  onAiImprove?: (currentHtml: string) => void;
  onAiTranslate?: (currentHtml: string, lang: string) => void;
  aiLoading?: boolean;
  className?: string;
}

export function DescriptionEditor({
  value,
  onChange,
  placeholder,
  onAiGenerate,
  onAiImprove,
  onAiTranslate,
  aiLoading = false,
  className,
}: DescriptionEditorProps) {
  const t = useTranslations("editor");
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        bold: false,
        italic: false,
        strike: false,
        code: false,
        codeBlock: false,
        blockquote: false,
        horizontalRule: false,
        hardBreak: false,
      }),
      Placeholder.configure({ placeholder: placeholder ?? t("placeholder") }),
    ],
    content: value,
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML());
    },
    editorProps: {
      attributes: {
        class:
          "prose prose-sm max-w-none min-h-[200px] p-3 focus:outline-none [&_h1]:text-xl [&_h1]:font-bold [&_h1]:mb-2 [&_h2]:text-lg [&_h2]:font-semibold [&_h2]:mb-2 [&_p]:mb-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_ol]:mb-2 [&_li]:mb-1",
      },
    },
  });

  if (!editor) return null;

  const toolbarButtons = [
    {
      icon: Heading1,
      label: t("heading1"),
      action: () => editor.chain().focus().toggleHeading({ level: 1 }).run(),
      active: editor.isActive("heading", { level: 1 }),
    },
    {
      icon: Heading2,
      label: t("heading2"),
      action: () => editor.chain().focus().toggleHeading({ level: 2 }).run(),
      active: editor.isActive("heading", { level: 2 }),
    },
    {
      icon: Pilcrow,
      label: t("paragraph"),
      action: () => editor.chain().focus().setParagraph().run(),
      active: editor.isActive("paragraph") && !editor.isActive("heading"),
    },
    {
      icon: List,
      label: t("bulletList"),
      action: () => editor.chain().focus().toggleBulletList().run(),
      active: editor.isActive("bulletList"),
    },
    {
      icon: ListOrdered,
      label: t("numberedList"),
      action: () => editor.chain().focus().toggleOrderedList().run(),
      active: editor.isActive("orderedList"),
    },
  ];

  return (
    <div className={cn("rounded-md border", className)}>
      <div className="flex items-center gap-1 border-b px-2 py-1">
        {toolbarButtons.map(({ icon: Icon, label, action, active }) => (
          <Button
            key={label}
            type="button"
            variant="ghost"
            size="sm"
            className={cn("h-8 w-8 p-0", active && "bg-accent")}
            onClick={action}
            title={label}
          >
            <Icon className="h-4 w-4" />
          </Button>
        ))}

        <div className="ml-auto">
          {(onAiGenerate || onAiImprove || onAiTranslate) && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={aiLoading}
                >
                  {aiLoading ? (
                    <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                  ) : (
                    <Sparkles className="mr-1 h-4 w-4" />
                  )}
                  AI
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {onAiGenerate && (
                  <DropdownMenuItem onClick={onAiGenerate}>
                    {t("aiGenerate")}
                  </DropdownMenuItem>
                )}
                {onAiImprove && (
                  <DropdownMenuItem
                    onClick={() => onAiImprove(editor.getHTML())}
                  >
                    {t("aiImprove")}
                  </DropdownMenuItem>
                )}
                {onAiTranslate && (
                  <>
                    <DropdownMenuItem
                      onClick={() => onAiTranslate(editor.getHTML(), "pl")}
                    >
                      {t("translateToPolish")}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={() => onAiTranslate(editor.getHTML(), "en")}
                    >
                      {t("translateToEnglish")}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={() => onAiTranslate(editor.getHTML(), "de")}
                    >
                      {t("translateToGerman")}
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}

function escapeHTML(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function plainTextToHTML(text: string): string {
  if (!text) return "";
  return text
    .split(/\n\n+/)
    .filter((p) => p.trim())
    .map((p) => `<p>${escapeHTML(p.replace(/\n/g, " ").trim())}</p>`)
    .join("");
}
