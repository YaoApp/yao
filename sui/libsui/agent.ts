/**
 * Yao AI Agent Pure JavaScript SDK
 * @author Max<max@iqka.com>
 * @maintainer https://yaoapps.com
 */

class Agent {
  private host: string;
  private token: string;
  private events: Record<AgentEvent, Handler>;
  private assistant_id?: string;
  private chat_id?: string;

  /**
   * Agent constructor
   * @param option Agent initialization options
   */
  constructor(option: AgentOption) {
    this.host = option.host || "/__yao/neo";
    this.token = option.token;
    this.events = {} as Record<AgentEvent, Handler>;
    this.assistant_id = option.assistant_id;
    this.chat_id = option.chat_id;
  }

  /**
   * Generate a chat ID
   * @returns A unique chat ID in the format of chat_[timestamp]_[random]
   */
  private makeChatID(): string {
    const random = Math.random().toString(36).substring(2, 15);
    const ts = Date.now();
    return `chat_${ts}_${random}`;
  }

  /**
   * Register an event handler
   * @param event Event type to listen for ("message" or "done")
   * @param handler Function to handle the event
   * @returns The Agent instance for chaining
   */
  On(event: AgentEvent, handler: Handler): Agent {
    this.events[event] = handler;
    return this;
  }

  /**
   * Call the AI Agent
   * @param id Agent ID
   * @param input Text message or input object with text and optional attachments
   * @param args Additional arguments to pass to the agent
   */
  async Call(id: string, input: AgentInput, ...args: any[]) {
    // Process input content
    let content: AgentInputContent;
    if (typeof input === "string") {
      content = { text: input };
    } else {
      content = { text: input.text };
      if (input.attachments && input.attachments.length > 0) {
        content.attachments = input.attachments.map((attachment) => ({
          name: attachment.name,
          url: attachment.url,
          type: attachment.type,
          content_type: attachment.content_type,
          bytes: attachment.bytes,
          created_at: attachment.created_at,
          file_id: attachment.file_id,
          chat_id: attachment.chat_id,
          assistant_id: attachment.assistant_id,
          description: attachment.description,
        }));
      }
    }

    const contentRaw = encodeURIComponent(JSON.stringify(content));
    const contextRaw = encodeURIComponent(JSON.stringify(args));
    const token = this.token;
    const assistantParam = this.assistant_id
      ? `&assistant_id=${this.assistant_id}`
      : "";
    const chatId = this.chat_id || this.makeChatID();

    const status_endpoint = `${this.host}/status?content=${contentRaw}&context=${contextRaw}&token=${token}&chat_id=${chatId}${assistantParam}`;
    const endpoint = `${this.host}?content=${contentRaw}&context=${contextRaw}&token=${token}&chat_id=${chatId}${assistantParam}`;

    const handleError = async (error: any) => {
      try {
        const response = await fetch(status_endpoint, {
          credentials: "include",
          headers: { Accept: "application/json" },
        });

        if (response.status === 200 || response.status === 201) return;

        const data = await response.json().catch(() => ({
          message: `HTTP ${response.status}`,
        }));

        let errorMessage = "Network error, please try again later";
        if (data?.message) {
          errorMessage = data.message;
        } else if (error.message?.includes("401")) {
          errorMessage = "Session expired: Please login again";
        } else if (error.message?.includes("403")) {
          errorMessage =
            "Access denied: Please check your permissions or login again";
        } else if (error.message?.includes("500")) {
          errorMessage = "Server error: The service is temporarily unavailable";
        } else if (error.message?.includes("404")) {
          errorMessage =
            "AI service not found: Please check your configuration";
        } else if (error.name === "TypeError") {
          errorMessage =
            "Connection failed: Please check your network connection";
        }

        const messageHandler = this.events["message"];
        if (messageHandler) {
          messageHandler({
            text: errorMessage,
            type: "error",
            is_neo: true,
            done: true,
          });
        }
      } catch (statusError) {
        const messageHandler = this.events["message"];
        if (messageHandler) {
          messageHandler({
            text: "Service unavailable, please try again later",
            type: "error",
            is_neo: true,
            done: true,
          });
        }
      }
    };

    try {
      const es = new EventSource(endpoint, {
        withCredentials: true,
      });

      // Track assistant information across messages
      const last_assistant: {
        assistant_id: string | null;
        assistant_name: string | null;
        assistant_avatar: string | null;
      } = {
        assistant_id: null,
        assistant_name: null,
        assistant_avatar: null,
      };

      let content = "";
      let last_type: string | null = null;

      es.onopen = () => {
        const messageHandler = this.events["message"];
        if (messageHandler) {
          messageHandler({
            text: "",
            is_neo: true,
            new: true,
          });
        }
      };

      es.onmessage = ({ data }: { data: string }) => {
        try {
          const formated_data = JSON.parse(data);
          if (!formated_data) return;

          const messageHandler = this.events["message"];
          if (!messageHandler) return;

          const {
            tool_id,
            begin,
            end,
            text,
            props,
            type,
            done,
            assistant_id,
            assistant_name,
            assistant_avatar,
            new: is_new,
            delta,
          } = formated_data;

          // Handle action message type
          if (type === "action") {
            const { namespace, primary, data_item, action, extra } =
              props || {};
            if (action && Array.isArray(action)) {
              messageHandler({
                text: text || "",
                type: "action",
                props: {
                  namespace: namespace || "chat",
                  primary: primary || "id",
                  data_item: data_item || {},
                  action,
                  extra,
                },
                is_neo: true,
                done: !!done,
              });

              if (done) {
                const doneHandler = this.events["done"];
                if (doneHandler) {
                  doneHandler({
                    text: text || "",
                    type: "action",
                    done: true,
                    is_neo: true,
                  });
                }
                es.close();
              }
              return;
            }
          }

          // Update content based on message properties
          if (text) {
            if (delta) {
              content = content + text;
              if (text?.startsWith("\r") || is_new) {
                content = text.replace("\r", "");
              }
            } else {
              content = text || "";
            }
          }

          // Update assistant information
          if (assistant_id) {
            last_assistant.assistant_id = assistant_id;
          }
          if (assistant_name) {
            last_assistant.assistant_name = assistant_name;
          }
          if (assistant_avatar) {
            last_assistant.assistant_avatar = assistant_avatar;
          }

          // Prepare message data
          const message_data: any = {
            ...formated_data,
            text: content,
            assistant_id: last_assistant.assistant_id || undefined,
            assistant_name: last_assistant.assistant_name || undefined,
            assistant_avatar: last_assistant.assistant_avatar || undefined,
          };

          // Handle tool and think message types
          if ((type === "tool" || type === "think") && delta) {
            message_data.type = "text";
            message_data.props = {
              ...(message_data.props || {}),
              id: tool_id,
              begin,
              end,
            };

            // Add closing tag if needed
            if (!content.includes(`</${type}>`)) {
              message_data.text = `${content}</${type}>`;
            }
          }

          // Send message to handler
          messageHandler(message_data);

          // Handle done event
          if (done) {
            const doneHandler = this.events["done"];
            if (doneHandler) {
              doneHandler({
                text: content,
                done: true,
                is_neo: true,
                type: message_data.type,
                props: message_data.props,
                assistant_id: last_assistant.assistant_id || undefined,
                assistant_name: last_assistant.assistant_name || undefined,
                assistant_avatar: last_assistant.assistant_avatar || undefined,
              });
            }
            es.close();
          }

          last_type = type || last_type;
        } catch (err) {
          console.error("Failed to parse message:", err);
        }
      };

      es.onerror = (ev) => {
        handleError(ev);
        es.close();
      };
    } catch (error) {
      handleError(error);
    }
  }
}

/**
 * Event types that can be listened to
 */
type AgentEvent = "message" | "done";

/**
 * Attachment information for file uploads
 */
interface AgentAttachment {
  name: string;
  url: string;
  type: string;
  content_type: string;
  bytes: number;
  created_at: string;
  file_id: string;
  chat_id?: string;
  assistant_id?: string;
  description?: string;
}

/**
 * Input content structure for agent calls
 */
interface AgentInputContent {
  text: string;
  attachments?: AgentAttachment[];
}

/**
 * Input type for agent calls, can be either a string or a structured input
 */
type AgentInput = string | AgentInputContent;

/**
 * Agent initialization options
 */
interface AgentOption {
  host?: string;
  token: string;
  assistant_id?: string;
  chat_id?: string;
}

/**
 * Message structure for agent responses
 */
interface AgentMessage {
  text: string;
  type?: string;
  done?: boolean;
  is_neo?: boolean;
  assistant_id?: string;
  assistant_name?: string;
  assistant_avatar?: string;
  props?: Record<string, any>;
  tool_id?: string;
  new?: boolean;
  delta?: boolean;
}

/**
 * Event handler function type
 */
interface Handler {
  (message: AgentMessage): void;
}

/**
 * Options for agent call configuration
 */
interface AgentCallOption {
  model: string;
  prompt: string;
  temperature: number;
  max_tokens: number;
}
