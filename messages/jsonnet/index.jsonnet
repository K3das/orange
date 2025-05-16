local utils = import "utils.libsonnet";
local colors = import "colors.libsonnet";

local setting_names = {
    "asr": "ASR",
};

local uses_cloudflare = "This feature uses Cloudflare for generating transcriptions ([privacy policy](https://www.cloudflare.com/privacypolicy/)), and your voice messages and transcriptions are never stored.";

{
    user_settings(ctx): {
        embeds: [
            {
                color: colors.orange,
                title: "Orange user preferences",
                fields: [
                    {
                        name: (if ctx.user_settings.asr_enabled then ":white_check_mark:" else ":x:") + " ASR",
                        value: "Enabling ASR will have Orange automatically transcribe your voice messages when you send them in chat, replying with the transcription. "+uses_cloudflare
                    }
                ]
            }
        ],
        components: [
            {
                type: 1, // action row
                components: [
                    if !ctx.user_settings.asr_enabled then {
                        type: 2,
                        label: "Enable ASR",
                        style: 1,
                        custom_id: ctx.user_settings.asr_enable_component_id
                    } else {
                        type: 2,
                        label: "Disable ASR",
                        style: 4,
                        custom_id: ctx.user_settings.asr_disable_component_id
                    },
                ]
            }
        ]
    },
    user_settings_toggle_response(ctx): 
        local toggle = ctx.user_settings_toggle_response;
        local setting_name = if toggle.setting in setting_names then setting_names[toggle.setting] else std.format("`%s`", toggle.setting);
        {
            embeds: [
                if toggle.changed then
                    if toggle.enabled && toggle.setting == "asr" then
                    {
                        color: colors.green,
                        title: "ASR Enabled",
                        description: "When you send a voice message, Orange will reply with an automatically generated transcription of your message."
                    } else {
                        color: colors.green,
                        title: std.format("%s %s", [if toggle.enabled then "Enabled" else "Disabled", toggle.setting])
                    }
                else
                    {
                        color: colors.green,
                        title: std.format("%s already %s", [setting_name, if toggle.enabled then "enabled" else "disabled"]),
                        description: std.format("Use </settings:%s> to update your preferences.", ctx.registered_commands["settings"].id)
                    }
            ]
        },
    interaction_error(ctx): {
        embeds: [
            {
                color: colors.red,
                title: "Error running interaction",
                description: ctx.interaction_error.message
            }
        ]
    },
    command_error(ctx): {
        embeds: [
            {
                color: colors.red,
                title: "Error running command",
                description: ctx.command_error.message
            }
        ]
    },
    command_create_hook_response(ctx): 
        local hook = ctx.command_create_hook_response.hook;
        {
            embeds: [
                {
                    color: colors.green,
                    title: "Created webhook",
                    description: std.format(|||
                        `https://discord.com/api/webhooks/%s/%s`
                        
                        -# You will only see this once. To regenerate, delete the webhook and re-run this command.
                    |||, [hook.id, hook.token])
                }
            ]
        },
    asr_progress(ctx): {
        embeds: [
            {
                color: colors.orange,
                title: "<a:tiger_spin:1370687556173172737> Transcribing voice message...",
            }
        ]
    },
    asr_result(ctx):
        local message = ctx.asr_result.caller_message;
        local icon_url = if !utils.zeroOrNull(message.member.avatar) then 
            std.format("https://cdn.discordapp.com/guilds/%s/users/%s/avatars/%s.png", [message.guild_id, message.author.id, message.member.avatar])
        else if !utils.zeroOrNull(message.author.avatar) then
            std.format("https://cdn.discordapp.com/avatars/%s/%s", [message.author.id, message.author.avatar])
        else
            null;

        local author = {
            icon_url: icon_url,
            name: if !utils.zeroOrNull(message.member.nick) then
                message.member.nick
            else if !utils.zeroOrNull(message.author.global_name) then
                message.author.global_name
            else 
                message.author.username
        };
        {
            embeds: [
                {
                    color: colors.orange,
                    description: ctx.asr_result.text,
                    footer: {
                        text: std.format("Transcribed by Orange in %.2f s", ctx.asr_result.duration)
                    },
                    author: author,
                }
            ]
        },
    asr_nudge(ctx): 
        local guild = ctx.asr_nudge.guild;
        {
            content: std.format(|||
                Hi there! It looks like you just sent a voice message in <#%(channel_id)s>.

                To make it easier for people to follow along (and to keep the server accessible to everyone), this bot offers automatic transcriptions (ASR) of voice messages. Opt-in using the button below, and you can run </settings:%(settings_command_id)s> to update your preferences at any time.
                
                - %(uses_cloudflare)s
                - If you choose not to enable ASR, we ask that you provide your own transcriptions your voice messages when possible.

                \- Mia
                -# You're receiving this one-off message because you're a member of **%(guild_name)s**.
            |||, {
                uses_cloudflare: uses_cloudflare,
                channel_id: ctx.asr_nudge.channel_id,
                guild_name: guild.name,
                settings_command_id: ctx.registered_commands["settings"].id
            }),
            components: [
                {
                    type: 1, // action row
                    components: [
                        {
                            type: 2,
                            label: "Enable ASR for future messages",
                            style: 1,
                            custom_id: ctx.asr_nudge.asr_enable_component_id
                        }
                    ]
                }
            ]
        },
    asr_error(ctx): {
        embeds: [
            {
                color: colors.red,
                title: "Error running transcription",
                description: ctx.asr_error.message
            }
        ]
    }
}