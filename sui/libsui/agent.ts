/**
 * Yao AI Agent Pure JavaScript SDK
 * @author Max<max@iqka.com>
 * @maintainer https://yaoapps.com
 */

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
  result?: any;
  previous_assistant_id?: string;
}

/**
 * Done event data structure
 */
type AgentDoneData = AgentMessage[];

/**
 * Event handler function types
 */
interface MessageHandler {
  (message: AgentMessage): void;
}

interface DoneHandler {
  (messages: AgentDoneData): void;
}

/**
 * Event types that can be listened to
 */
type AgentEvent = "message" | "done";

/**
 * Event handlers record type
 */
interface EventHandlers {
  message?: MessageHandler;
  done?: DoneHandler;
}

class Agent {
  private host: string;
  private token: string;
  private events: EventHandlers;
  private assistant_id: string;
  private chat_id?: string;
  private es: EventSource | null;
  private context: Record<string, any>;
  private silent?: boolean = false;
  private history_visible?: boolean = false;

  /**
   * Agent constructor
   * @param option Agent initialization options
   */
  constructor(assistant_id: string, option: AgentOption) {
    this.host = option.host || "/api/__yao/neo";
    this.token = option.token;
    this.events = {};
    this.assistant_id = assistant_id;
    this.chat_id = option.chat_id;
    this.es = null;
    this.context = option.context || {};

    // Set silent mode, default is true
    if (option.silent !== undefined) {
      this.silent =
        option.silent === true ||
        option.silent === "true" ||
        option.silent === 1 ||
        option.silent === "1"
          ? false
          : true;
    }

    // Set history visible mode, default is false
    if (option.history_visible !== undefined) {
      this.history_visible =
        option.history_visible === true ||
        option.history_visible === "true" ||
        option.history_visible === 1 ||
        option.history_visible === "1"
          ? true
          : false;
    }
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
  On<E extends AgentEvent>(
    event: E,
    handler: E extends "message" ? MessageHandler : DoneHandler
  ): Agent {
    if (event === "message") {
      this.events.message = handler as MessageHandler;
    } else if (event === "done") {
      this.events.done = handler as DoneHandler;
    }
    return this;
  }

  /**
   * Cancel the agent
   */
  Cancel() {
    if (this.es) {
      this.es.close();
      this.es = null;
    }
  }

  /**
   * Call the AI Agent
   * @param input Text message or input object with text and optional attachments
   * @param args Additional arguments to pass to the agent
   */
  async Call(input: AgentInput, ...args: any[]): Promise<any> {
    return new Promise((resolve, reject) => {
      const messages: AgentMessage[] = [];
      let lastAssistant = {
        assistant_id: null as string | null,
        assistant_name: null as string | null,
        assistant_avatar: null as string | null,
      };

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

      // Add context to the content
      const context = { ...this.context, args };
      const contentRaw = encodeURIComponent(JSON.stringify(content));
      const contextRaw = encodeURIComponent(JSON.stringify(context));
      const token = this.token;
      const silent = this.silent ? "true" : "false";
      const history_visible = this.history_visible ? "true" : "false";
      const chatId = this.chat_id || this.makeChatID();
      const assistantParam = `&assistant_id=${this.assistant_id}`;
      const status_endpoint = `${this.host}/status?content=${contentRaw}&context=${contextRaw}&token=${token}&chat_id=${chatId}${assistantParam}`;
      const endpoint = `${this.host}?client_type=jssdk&content=${contentRaw}&context=${contextRaw}&token=${token}&silent=${silent}&history_visible=${history_visible}&chat_id=${chatId}${assistantParam}`;

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
            errorMessage =
              "Server error: The service is temporarily unavailable";
          } else if (error.message?.includes("404")) {
            errorMessage =
              "AI service not found: Please check your configuration";
          } else if (error.name === "TypeError") {
            errorMessage =
              "Connection failed: Please check your network connection";
          }

          const messageHandler = this.events["message"] as MessageHandler;
          if (messageHandler) {
            messageHandler({
              text: errorMessage,
              type: "error",
              is_neo: true,
              done: true,
            });
          }

          return reject(new Error(errorMessage));
        } catch (statusError) {
          const messageHandler = this.events["message"] as MessageHandler;
          if (messageHandler) {
            messageHandler({
              text: "Service unavailable, please try again later",
              type: "error",
              is_neo: true,
              done: true,
            });
          }

          return reject(
            new Error("Service unavailable, please try again later")
          );
        }
      };

      try {
        let last_type: string | null = null;
        const es = new EventSource(endpoint, { withCredentials: true });
        this.es = es;

        es.onopen = () => {};
        es.onmessage = ({ data }: { data: string }) => {
          try {
            const formated_data = JSON.parse(data);
            if (!formated_data) return;

            const messageHandler = this.events["message"] as MessageHandler;
            if (!messageHandler) return;

            const {
              tool_id,
              begin,
              type,
              end,
              text,
              props,
              done,
              assistant_id,
              assistant_name,
              assistant_avatar,
              new: is_new,
              delta,
              result,
            } = formated_data;

            // Handle action message type
            if (type === "action") {
              const { namespace, primary, data_item, action, extra } =
                props || {};
              if (action && Array.isArray(action)) {
                const actionMessage = {
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
                };

                messages.push(actionMessage);
                messageHandler(actionMessage);

                if (done) {
                  const doneHandler = this.events["done"] as DoneHandler;
                  doneHandler?.(messages);
                  es.close();
                }
                return resolve(result);
              }
            }

            // Check if we need to create a new message
            const shouldCreateNewMessage =
              (type !== last_type &&
                (!done || (done === true && (text || props)))) || // if type changed or done is true and there is text or props
              messages.length === 0 ||
              (assistant_id &&
                messages[messages.length - 1].assistant_id !== assistant_id) ||
              (is_new && !delta); // Only create new message if it's new and not a delta update

            // Update last type
            last_type = type;

            // Update assistant information
            if (assistant_id) lastAssistant.assistant_id = assistant_id;
            if (assistant_name) lastAssistant.assistant_name = assistant_name;
            if (assistant_avatar)
              lastAssistant.assistant_avatar = assistant_avatar;

            if (shouldCreateNewMessage) {
              // Mark the last message as done if it exists
              if (messages.length > 0 && messages[messages.length - 1].is_neo) {
                messages[messages.length - 1] = {
                  ...messages[messages.length - 1],
                  done: true,
                };
              }

              // Create new message with all original properties
              const newMessage = {
                text: text || "",
                type: type || "text",
                props,
                is_neo: true,
                new: is_new, // Only set new if it's from the original message
                tool_id,
                result: result,
                assistant_id: lastAssistant.assistant_id || undefined,
                assistant_name: lastAssistant.assistant_name || undefined,
                assistant_avatar: lastAssistant.assistant_avatar || undefined,
              };

              messages.push(newMessage);
              messageHandler(newMessage);

              // If the message is done, close the event source
              if (done) {
                const doneHandler = this.events["done"] as DoneHandler;
                doneHandler?.(messages);
                es.close();
                return resolve(result);
              }
              return;
            }

            // Get current message (we know it exists because we checked messages.length above)
            const current_answer = messages[messages.length - 1];

            // Set previous assistant id
            if (messages.length > 1) {
              const previous_message = messages[messages.length - 2];
              if (previous_message.assistant_id) {
                current_answer.previous_assistant_id =
                  previous_message.assistant_id;
              }
            }

            // Handle message completion (done flag is set)
            if (done) {
              if (text) {
                current_answer.text = text;
              }
              if (type) {
                current_answer.type = type;
              }
              if (props) {
                current_answer.props = props;
              }

              // Set result if available
              if (result) {
                current_answer.result = result;
              }

              // Mark all previous neo messages as done
              for (let i = messages.length - 1; i >= 0; i--) {
                const message = messages[i];
                if (message.is_neo) {
                  if (message.done) break;
                  messages[i] = { ...message, done: true };
                }
              }

              const doneHandler = this.events["done"] as DoneHandler;
              doneHandler?.(messages);
              es.close();
              return resolve(result);
            }

            // Skip processing if no content to update
            if (!text && !props && !type) return;

            // Update props if available
            if (props) {
              if (type === "think" || type === "tool") {
                current_answer.props = {
                  ...(current_answer.props || {}),
                  id: tool_id,
                  begin,
                  end,
                };
              } else {
                current_answer.props = props;
              }
            }

            // Handle text content
            if (text) {
              if (delta) {
                current_answer.text = (current_answer.text || "") + text;
                if (text.startsWith("\r")) {
                  current_answer.text = text.replace("\r", "");
                }
              } else {
                current_answer.text = text;
              }
            }

            // Send current message to handler
            messageHandler(current_answer);
          } catch (err) {
            const errorMessage =
              err.message || JSON.stringify(err) || "未知错误";
            console.error("Failed to parse message:", err);
            reject(new Error(errorMessage));
          }
        };

        es.onerror = (ev) => {
          handleError(ev);
          es.close();
        };
      } catch (error) {
        handleError(error);
      }
    });
  }
}

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
type AgentInput =
  | string
  | {
      text: string;
      attachments?: AgentAttachment[];
    };

/**
 * Agent initialization options
 */
interface AgentOption {
  host?: string;
  token: string;
  silent?: boolean | string | number;
  history_visible?: boolean | string | number;
  chat_id?: string;
  context?: Record<string, any>;
}
