import { createCrudHooks } from "./create-crud-hooks";
import type {
  MessageTemplate,
  CreateMessageTemplateRequest,
  UpdateMessageTemplateRequest,
  MessageTemplateListParams,
} from "@/types/api";

const hooks = createCrudHooks<
  MessageTemplate,
  CreateMessageTemplateRequest,
  UpdateMessageTemplateRequest,
  MessageTemplateListParams
>({
  resourceKey: "message-templates",
  basePath: "/v1/message-templates",
  updateMethod: "PUT",
});

export const useMessageTemplates = hooks.useList;
export const useMessageTemplate = hooks.useGet;
export const useCreateMessageTemplate = hooks.useCreate;
export const useUpdateMessageTemplate = hooks.useUpdate;
export const useDeleteMessageTemplate = hooks.useDelete;
