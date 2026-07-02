// ANSI parser to plain HTML with CSS colored spans
function parseAnsi(text) {
    return text
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('\u001b[38;5;99m', '<span class="t-purple">')
        .replaceAll('\u001b[38;5;120m', '<span class="t-green">')
        .replaceAll('\u001b[48;5;244m', '<span class="t-bg-gray">')
        .replaceAll('\u001b[31;1m', '<span class="t-red">')
        .replaceAll('\u001b[35;1m', '<span class="t-magenta">')
        .replaceAll('\u001b[90m', '<span class="t-gray">')
        .replaceAll('\u001b[0m', '</span>');
}

document.addEventListener('DOMContentLoaded', () => {
    // Commands data
    const commands = {
        add: {
            input: 'zenith task add "Launch Zenith CLI website" --priority critical',
            output: `Task added with ID: 12\nLog created: [2026-07-02 15:12:04] CREATED - Task created: 'Launch Zenith CLI website'`
        },
        summary: {
            input: 'zenith summary',
            output: `\u001b[38;5;99m Zenith Daily Summary \u001b[0m\nToday is \u001b[38;5;99mThursday, 02 Jul 2026\u001b[0m\n\n\u001b[38;5;99mProject Tasks:\u001b[0m\n \u001b[48;5;244m Unassigned \u001b[0m\n  [ ] Learn Go Bubble Tea (0h 45m 0s) 📅 03 Jul\n  [x] Install Ollama local models (1h 10m 0s)\n  [ ] Launch Zenith CLI website (0h 0m 0s) \u001b[31;1m!!\u001b[0m\n\n\u001b[38;5;120mDaily Habits:\u001b[0m\n  • \u001b[38;5;120mMorning Run\u001b[0m (daily)\n  • \u001b[38;5;120mRead Book\u001b[0m (daily)`
        },
        timer: {
            input: 'zenith timer add 25m "Write implementation plan" --task 12',
            output: `Timer scheduled! ID: 4. Will trigger in 25m (at 15:37:04).\nPassive tracking enabled. Desktop notification queued.`
        },
        ai: {
            input: 'zenith ai suggest',
            output: `\u001b[38;5;99m Zenith AI Suggestion \u001b[0m\nAnalyzing tasks and habits... Please wait.\n\n### 🎯 Task Recommendations\n1. **Launch Zenith CLI website** (Priority: CRITICAL): Focus on this first. Having a stunning visual presence is important for onboarding users.\n2. **Learn Go Bubble Tea** (Priority: MEDIUM): Dedicate 30 mins to clean up the TUI widgets.\n\n### 📅 Suggested Schedule\n- **15:30 - 17:00**: Work on Zenith CLI website.\n- **17:00 - 17:30**: Learn Go Bubble Tea.\n\n### 💪 Habit Feedback\nYou logged all daily habits yesterday. Keep the streak going! 🚀`
        },
        tui: {
            input: 'zenith tui',
            output: `┌────────────────────────────────────────────────────────────┐\n│ \u001b[38;5;99m Zenith TUI Dashboard \u001b[0m   📅 \u001b[35;1mThursday, 02 Jul 2026\u001b[0m         │\n│                                                            │\n│ Progress: 1/3 completed [\u001b[38;5;120m█████░░░░░░░░░░\u001b[0m] 33%              │\n│                                                            │\n│ > [ ] Learn Go Bubble Tea (0h 45m 0s)                      │\n│   [x] Install Ollama local models (1h 10m 0s)              │\n│   [ ] Launch Zenith CLI website \u001b[31;1m!!\u001b[0m (0h 0m 0s ⏳)          │\n│                                                            │\n│ \u001b[90m(j/k: move, enter: toggle, s: timer, q: quit)\u001b[0m              │\n└────────────────────────────────────────────────────────────┘`
        }
    };

    const typingElement = document.querySelector('.typing-demo');
    const outputElement = document.querySelector('.term-output-demo');
    const demoButtons = document.querySelectorAll('.btn-demo');
    
    let typingInterval = null;

    function typeCommand(key) {
        // Clear active typing
        if (typingInterval) {
            clearInterval(typingInterval);
        }
        
        typingElement.textContent = '';
        outputElement.innerHTML = '';

        const cmdText = commands[key].input;
        const cmdOutput = commands[key].output;
        let index = 0;

        typingInterval = setInterval(() => {
            if (index < cmdText.length) {
                typingElement.textContent += cmdText[index];
                index++;
            } else {
                clearInterval(typingInterval);
                typingInterval = null;
                // Add simulated delay before output
                setTimeout(() => {
                    outputElement.innerHTML = parseAnsi(cmdOutput);
                }, 200);
            }
        }, 40);
    }

    // Add click listeners to buttons
    demoButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            // Remove active class
            demoButtons.forEach(b => b.classList.remove('active'));
            
            // Add active class to clicked button
            const key = btn.dataset.cmd;
            btn.classList.add('active');
            
            typeCommand(key);
        });
    });

    // Run first demo command on load
    typeCommand('add');
});
