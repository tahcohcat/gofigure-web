// web/static/js/app.js - Updated with user features

class MysteryGame {
    constructor() {
        this.currentSession = null;
        this.selectedCharacter = null;
        this.characterStressLevels = {};
        this.ttsEnabled = true;
        this.hintsEnabled = true;
        this.currentUser = null;

        this.init();
    }

    async init() {
        await this.loadMysteries();
        this.setupEventListeners();
        this.checkAuthStatus();
    }

    async checkAuthStatus() {
        try {
            const response = await fetch('/profile', {
                headers: { 'Accept': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                this.currentUser = data.user;
                this.showUserInfo();
            }
        } catch (error) {
            console.log('Not authenticated or error checking auth:', error);
        }
    }

    showUserInfo() {
        if (this.currentUser) {
            // Add user info to header
            const header = document.querySelector('header');
            const userInfo = document.createElement('div');
            userInfo.className = 'user-info';
            userInfo.innerHTML = `
                <span>Welcome, ${this.currentUser.display_name}! 🕵️</span>
                <a href="/profile" class="btn btn-secondary btn-sm">Profile</a>
                <button onclick="this.logout()" class="btn btn-secondary btn-sm">Logout</button>
            `;
            header.appendChild(userInfo);
        }
    }

    async logout() {
        try {
            await fetch('/logout', { method: 'POST' });
            window.location.href = '/login';
        } catch (error) {
            console.error('Logout failed:', error);
        }
    }

    async loadMysteries() {
        try {
            const response = await fetch('/api/v1/mysteries');
            const data = await response.json();
            this.displayMysteries(data.mysteries);
        } catch (error) {
            console.error('Failed to load mysteries:', error);
        }
    }

    displayMysteries(mysteries) {
        const mysteryList = document.getElementById('mystery-list');
        mysteryList.innerHTML = '';

        mysteries.forEach(mystery => {
            const mysteryCard = document.createElement('div');
            mysteryCard.className = 'mystery-card';
            mysteryCard.innerHTML = `
                <h3>${mystery.title}</h3>
                <p>${mystery.description}</p>
                <div class="difficulty difficulty-${mystery.difficulty.toLowerCase()}">${mystery.difficulty}</div>
                <button class="btn btn-primary" onclick="game.startMystery('${mystery.id}')">
                    Start Investigation
                </button>
            `;
            mysteryList.appendChild(mysteryCard);
        });
    }

    async startMystery(mysteryId) {
        this.showScreen('loading-screen');

        try {
            const response = await fetch('/api/v1/game/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ mystery_id: mysteryId })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.currentSession = data.session_id;
            this.setupGame(data);
            this.startTimer();
        } catch (error) {
            console.error('Failed to start mystery:', error);
            alert('Failed to start mystery. Please try again.');
            this.showScreen('mystery-selection');
        }
    }

    setupGame(data) {
        document.getElementById('mystery-title').textContent = data.title;
        document.getElementById('intro-text').textContent = data.intro;

        // Setup characters with stress tracking
        this.displayCharacters(data.characters);
        this.characterStressLevels = {};
        data.characters.forEach(char => {
            this.characterStressLevels[char.name] = 0;
        });

        this.gameData = data;
        this.showScreen('game-screen');
    }

    displayCharacters(characters) {
        const charactersList = document.getElementById('characters-list');
        charactersList.innerHTML = '';

        characters.forEach(character => {
            const characterDiv = document.createElement('div');
            characterDiv.className = 'character-card';
            characterDiv.innerHTML = `
                <img src="/${character.sprite}" alt="${character.name}" class="character-avatar">
                <div class="character-info">
                    <h4>${character.name}</h4>
                    <div class="stress-indicator">
                        <div class="stress-bar">
                            <div class="stress-fill" id="stress-${character.name}" style="width: 0%"></div>
                        </div>
                        <span class="stress-label" id="stress-label-${character.name}">Calm</span>
                    </div>
                </div>
            `;

            characterDiv.addEventListener('click', () => this.selectCharacter(character));
            charactersList.appendChild(characterDiv);
        });
    }

    selectCharacter(character) {
        // Remove previous selection
        document.querySelectorAll('.character-card').forEach(card => {
            card.classList.remove('selected');
        });

        // Select current character
        event.currentTarget.classList.add('selected');
        this.selectedCharacter = character;

        // Enable question input
        document.getElementById('question-input').disabled = false;
        document.getElementById('ask-btn').disabled = false;
        document.getElementById('accuse-btn').disabled = false;

        // Show selected character
        document.getElementById('selected-character').classList.remove('hidden');
        document.getElementById('selected-character-name').textContent = character.name;
    }

    async askQuestion() {
        if (!this.selectedCharacter) {
            alert('Please select a character first');
            return;
        }

        const questionInput = document.getElementById('question-input');
        const question = questionInput.value.trim();

        if (!question) {
            alert('Please enter a question');
            return;
        }

        const askBtn = document.getElementById('ask-btn');
        askBtn.disabled = true;
        askBtn.textContent = 'Thinking...';

        try {
            const currentStress = this.characterStressLevels[this.selectedCharacter.name] || 0;

            const response = await fetch(`/api/v1/game/${this.currentSession}/ask`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    character_name: this.selectedCharacter.name,
                    question: question,
                    current_stress: currentStress
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.displayConversation(question, data);
            this.updateStress(data.character, data.stress_level, data.stress_state);

            // Clear question input
            questionInput.value = '';

            // Play TTS if enabled
            if (this.ttsEnabled) {
                await this.playTTS(data.response, data.character, data.emotion);
            }

        } catch (error) {
            console.error('Failed to ask question:', error);
            alert('Failed to get response. Please try again.');
        } finally {
            askBtn.disabled = false;
            askBtn.textContent = 'Ask';
        }
    }

    displayConversation(question, response) {
        const conversationHistory = document.getElementById('conversation-history');

        // Add question
        const questionDiv = document.createElement('div');
        questionDiv.className = 'message detective-message';
        questionDiv.innerHTML = `
            <strong>You:</strong> ${question}
        `;

        // Add response
        const responseDiv = document.createElement('div');
        responseDiv.className = `message character-message ${response.emotion}`;
        responseDiv.innerHTML = `
            <strong>${response.character}:</strong> ${response.response}
            <div class="message-meta">
                Emotion: ${response.emotion} | Stress: ${response.stress_state}
            </div>
        `;

        conversationHistory.appendChild(questionDiv);
        conversationHistory.appendChild(responseDiv);
        conversationHistory.scrollTop = conversationHistory.scrollHeight;
    }

    updateStress(characterName, stressLevel, stressState) {
        this.characterStressLevels[characterName] = stressLevel;

        const stressFill = document.getElementById(`stress-${characterName}`);
        const stressLabel = document.getElementById(`stress-label-${characterName}`);

        if (stressFill && stressLabel) {
            stressFill.style.width = `${stressLevel}%`;
            stressLabel.textContent = stressState;

            // Update color based on stress level
            stressFill.className = 'stress-fill';
            if (stressLevel < 25) stressFill.classList.add('stress-low');
            else if (stressLevel < 70) stressFill.classList.add('stress-medium');
            else stressFill.classList.add('stress-high');
        }
    }

    async playTTS(text, character, emotion) {
        try {
            const response = await fetch('/api/v1/tts/speak', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    text: text,
                    character: character,
                    emotion: emotion,
                    session_id: this.currentSession
                })
            });

            if (response.ok) {
                const audioBlob = await response.blob();
                const audio = new Audio(URL.createObjectURL(audioBlob));
                await audio.play();
            }
        } catch (error) {
            console.error('TTS playback failed:', error);
        }
    }

    showAccusationModal() {
        if (!this.gameData) return;

        const modal = document.getElementById('accusation-modal');
        const charactersDiv = document.getElementById('accusation-characters');

        charactersDiv.innerHTML = '';

        this.gameData.characters.forEach(character => {
            const button = document.createElement('button');
            button.className = 'btn btn-accusation';
            button.innerHTML = `
                <img src="/${character.sprite}" alt="${character.name}">
                <span>${character.name}</span>
            `;
            button.onclick = () => this.makeAccusation(character.name);
            charactersDiv.appendChild(button);
        });

        modal.classList.remove('hidden');
    }

    async makeAccusation(suspect) {
        try {
            const response = await fetch(`/api/v1/game/${this.currentSession}/accuse`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ suspect: suspect })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            this.showResult(result);

        } catch (error) {
            console.error('Failed to make accusation:', error);
            alert('Failed to submit accusation. Please try again.');
        }

        document.getElementById('accusation-modal').classList.add('hidden');
    }

    showResult(result) {
        const modal = document.getElementById('result-modal');
        const title = document.getElementById('result-title');
        const message = document.getElementById('result-message');

        title.textContent = result.correct ? '🎉 Case Solved!' : '❌ Case Unsolved';
        message.innerHTML = `
            ${result.message}<br><br>
            <strong>Solution:</strong><br>
            Killer: ${result.killer}<br>
            Weapon: ${result.weapon}<br>
            Location: ${result.location}<br><br>
            Time spent: ${Math.floor(result.time_spent / 60)}m ${result.time_spent % 60}s<br>
            Questions asked: ${result.questions}
        `;

        modal.classList.remove('hidden');

        // Stop timer
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
        }
    }

    startTimer() {
        this.timerInterval = setInterval(async () => {
            try {
                const response = await fetch(`/api/v1/game/${this.currentSession}/timer`);
                if (response.ok) {
                    const data = await response.json();
                    this.updateTimerDisplay(data.remaining_time);

                    if (data.game_over) {
                        clearInterval(this.timerInterval);
                        alert('⏰ Time\'s up! The case goes unsolved...');
                    }
                }
            } catch (error) {
                console.error('Timer update failed:', error);
            }
        }, 1000);
    }

    updateTimerDisplay(seconds) {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = seconds % 60;
        document.getElementById('timer').textContent =
            `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`;

        // Change color when running low
        const timerElement = document.getElementById('timer');
        if (seconds < 300) { // Less than 5 minutes
            timerElement.style.color = '#e74c3c';
        } else if (seconds < 600) { // Less than 10 minutes
            timerElement.style.color = '#f39c12';
        }
    }

    setupEventListeners() {
        // Game controls
        document.getElementById('start-investigation-btn').addEventListener('click', () => {
            document.getElementById('intro-section').classList.add('hidden');
            document.getElementById('investigation-section').classList.remove('hidden');
        });

        document.getElementById('ask-btn').addEventListener('click', () => this.askQuestion());

        document.getElementById('question-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.askQuestion();
        });

        document.getElementById('accuse-btn').addEventListener('click', () => this.showAccusationModal());

        // Modal controls
        document.getElementById('cancel-accusation').addEventListener('click', () => {
            document.getElementById('accusation-modal').classList.add('hidden');
        });

        document.getElementById('play-again-btn').addEventListener('click', () => {
            this.resetGame();
        });

        document.getElementById('back-to-menu-btn').addEventListener('click', () => {
            this.resetGame();
        });

        document.getElementById('back-btn').addEventListener('click', () => {
            this.resetGame();
        });

        // Settings toggles
        document.getElementById('tts-toggle').addEventListener('click', (e) => {
            this.ttsEnabled = !this.ttsEnabled;
            e.target.textContent = this.ttsEnabled ? '🔊 TTS On' : '🔇 TTS Off';
        });

        document.getElementById('hints-toggle').addEventListener('click', (e) => {
            this.hintsEnabled = !this.hintsEnabled;
            e.target.textContent = this.hintsEnabled ? '💡 Hints On' : '💡 Hints Off';
        });

        document.getElementById('timer-toggle-btn').addEventListener('click', async () => {
            if (this.currentSession) {
                try {
                    const response = await fetch(`/api/v1/game/${this.currentSession}/timer/toggle`, {
                        method: 'POST'
                    });
                    if (response.ok) {
                        const data = await response.json();
                        document.getElementById('timer-toggle-btn').textContent =
                            data.timer_enabled ? '⏳ Timer On' : '⏸️ Timer Off';
                    }
                } catch (error) {
                    console.error('Failed to toggle timer:', error);
                }
            }
        });

        document.getElementById('tts-test').addEventListener('click', async () => {
            try {
                const response = await fetch('/api/v1/tts/test');
                if (response.ok) {
                    const audioBlob = await response.blob();
                    const audio = new Audio(URL.createObjectURL(audioBlob));
                    await audio.play();
                } else {
                    alert('TTS test failed - check server configuration');
                }
            } catch (error) {
                console.error('TTS test failed:', error);
                alert('TTS test failed');
            }
        });
    }

    showScreen(screenId) {
        document.querySelectorAll('.screen').forEach(screen => {
            screen.classList.remove('active');
        });
        document.getElementById(screenId).classList.add('active');
    }

    resetGame() {
        this.currentSession = null;
        this.selectedCharacter = null;
        this.characterStressLevels = {};

        if (this.timerInterval) {
            clearInterval(this.timerInterval);
        }

        // Reset UI
        document.getElementById('conversation-history').innerHTML = '';
        document.getElementById('question-input').value = '';
        document.getElementById('question-input').disabled = true;
        document.getElementById('ask-btn').disabled = true;
        document.getElementById('accuse-btn').disabled = true;
        document.getElementById('selected-character').classList.add('hidden');
        document.getElementById('intro-section').classList.remove('hidden');
        document.getElementById('investigation-section').classList.add('hidden');
        document.getElementById('result-modal').classList.add('hidden');
        document.getElementById('timer').textContent = '60:00';
        document.getElementById('timer').style.color = '';

        this.showScreen('mystery-selection');
    }
}

// Initialize game when page loads
let game;
document.addEventListener('DOMContentLoaded', () => {
    game = new MysteryGame();
});