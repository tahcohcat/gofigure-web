class GoFigureApp {
    constructor() {
        this.currentSession = null;
        this.currentMystery = null;
        this.selectedCharacter = null;
        this.currentAudio = null;
        this.ttsEnabled = true;
        this.hintsEnabled = true;
        this.timerEnabled = true;
        this.timerInterval = null;

        this.characterStressLevels = {}; // Store stress levels for each character
        this.stressKeywords = {
            high: ['murder', 'kill', 'weapon', 'blood', 'death', 'guilty', 'lie', 'alibi'],
            medium: ['suspicious', 'secret', 'hidden', 'truth', 'why', 'when', 'where'],
            low: ['weather', 'family', 'work', 'hobby', 'general', 'hello', 'how']
        };

        this.init();
    }

    init() {
        this.bindEvents();
        this.loadMysteries();
    }

    bindEvents() {
        // Hints toggle
        document.getElementById('hints-toggle').addEventListener('click', () => {
            this.hintsEnabled = !this.hintsEnabled;
            document.getElementById('hints-toggle').textContent = this.hintsEnabled ? '💡 Hints On' : '💡 Hints Off';
            if (this.currentMystery) {
                this.renderCharacters(this.currentMystery.characters);
            }
        });

        document.getElementById('timer-toggle-btn').addEventListener('click', () => {
            this.toggleTimer();
        });

        // Mystery selection
        document.addEventListener('click', (e) => {
            const card = e.target.closest('.mystery-card');
            if (card) {
                const mysteryId = card.dataset.mysteryId;
                this.selectMystery(mysteryId);
            }
        });

        // Character selection
        document.addEventListener('click', (e) => {
            if (e.target.closest('.character-card')) {
                const card = e.target.closest('.character-card');
                const characterName = card.dataset.characterName;
                this.selectCharacter(characterName);
            }
        });

        // Navigation
        document.getElementById('back-btn').addEventListener('click', () => {
            this.resetGame();
            this.showScreen('mystery-selection');
        });

        document.getElementById('start-investigation-btn').addEventListener('click', () => {
            this.startInvestigation();
        });

        // Question asking
        document.getElementById('ask-btn').addEventListener('click', () => {
            this.askQuestion();
        });

        document.getElementById('question-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.target.disabled) {
                this.askQuestion();
            }
        });

        // Accusation functionality
        document.getElementById('accuse-btn').addEventListener('click', () => {
            this.showAccusationModal();
        });

        document.getElementById('cancel-accusation').addEventListener('click', () => {
            this.hideModal('accusation-modal');
        });

        // Result modal actions
        document.getElementById('play-again-btn').addEventListener('click', () => {
            this.resetGame();
            this.selectMystery(this.currentMystery.id || 'cruise_ship');
            this.hideModal('result-modal');
        });

        document.getElementById('back-to-menu-btn').addEventListener('click', () => {
            this.resetGame();
            this.showScreen('mystery-selection');
            this.hideModal('result-modal');
        });

        // TTS controls
        document.getElementById('tts-toggle').addEventListener('click', () => {
            const enabled = this.toggleTTS();
            document.getElementById('tts-toggle').textContent = enabled ? '🔊 TTS On' : '🔇 TTS Off';
        });

        document.getElementById('tts-test').addEventListener('click', () => {
            this.playTestTTS();
        });
    }

    async loadMysteries() {
        try {
            const response = await fetch('/api/v1/mysteries');
            const data = await response.json();
            
            this.renderMysteries(data.mysteries);
        } catch (error) {
            console.error('Failed to load mysteries:', error);
            this.showError('Failed to load mysteries. Please try again.');
        }
    }

    renderMysteries(mysteries) {
        const container = document.getElementById('mystery-list');
        container.innerHTML = '';

        mysteries.forEach(mystery => {
            const card = document.createElement('div');
            card.className = 'mystery-card';
            card.dataset.mysteryId = mystery.id;
            
            card.innerHTML = `
                <div class="mystery-header">
                    <h3>${mystery.title}</h3>
                    <span class="difficulty-badge difficulty-${mystery.difficulty.toLowerCase()}">${mystery.difficulty}</span>
                </div>
                <p>${mystery.description}</p>
                <small>Mystery ID: ${mystery.id}</small>
            `;
            
            container.appendChild(card);
        });
    }

    async selectMystery(mysteryId) {
        this.showScreen('loading-screen');
        
        try {
            const response = await fetch('/api/v1/game/start', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ mystery_id: mysteryId })
            });

            const data = await response.json();
            //console.log('Received mystery data:', JSON.stringify(data, null, 2));
            
            if (response.ok) {
                this.currentSession = data.session_id;
                this.currentMystery = data;
                this.showGameScreen(data);

                // Start the timer
                this.timerInterval = setInterval(() => {
                    this.updateTimer();
                }, 1000);
            } else {
                throw new Error(data.error || 'Failed to start game');
            }
        } catch (error) {
            console.error('Failed to start game:', error);
            this.showError('Failed to start the mystery. Please try again.');
            this.showScreen('mystery-selection');
        }
    }

    showGameScreen(mystery) {
        // Set title
        document.getElementById('mystery-title').textContent = mystery.title;
        
        // Set introduction
        document.getElementById('intro-text').textContent = mystery.intro;
        
        // Render characters
        this.renderCharacters(mystery.characters);
        
        // Show game screen
        this.showScreen('game-screen');
        
        // Hide investigation section initially
        document.getElementById('investigation-section').classList.add('hidden');
    }

    renderCharacters(characters) {
        const container = document.getElementById('characters-list');
        container.innerHTML = '';

        characters.forEach(character => {
            // Initialize stress level if not exists
            if (!this.characterStressLevels[character.name]) {
                this.characterStressLevels[character.name] = this.getInitialStressLevel(character);
            }

            const card = document.createElement('div');
            card.className = 'character-card';
            card.dataset.characterName = character.name;

            // Apply initial stress level
            this.applyStressLevel(card, this.characterStressLevels[character.name]);

            card.innerHTML = `
            <img src="${character.sprite}" alt="${character.name}" class="character-sprite">
            <div class="stress-indicators stress-tooltip" data-tooltip="Stress level indicator">
                ${this.generateStressBars()}
            </div>
            <div class="character-info">
                <h4>${character.name}</h4>
                ${this.hintsEnabled ? `<p>${character.personality}</p>` : ''}
                <div class="stress-status">
                    <span class="stress-label">${this.getStressLabel(this.characterStressLevels[character.name])}</span>
                </div>
            </div>
        `;

            container.appendChild(card);
        });
    }

    // Generate stress bar HTML
    generateStressBars() {
        return `
        <div class="stress-bar"><div class="stress-fill"></div></div>
        <div class="stress-bar"><div class="stress-fill"></div></div>
        <div class="stress-bar"><div class="stress-fill"></div></div>
        <div class="stress-bar"><div class="stress-fill"></div></div>
        <div class="stress-bar"><div class="stress-fill"></div></div>
    `;
    }

// Get initial stress level based on character personality
    getInitialStressLevel(character) {
        // Base stress on personality keywords
        const personality = character.personality.toLowerCase();

        if (personality.includes('nervous') || personality.includes('anxious') || personality.includes('secretive')) {
            return 40 + Math.random() * 20; // 40-60
        } else if (personality.includes('calm') || personality.includes('composed') || personality.includes('professional')) {
            return 10 + Math.random() * 15; // 10-25
        } else if (personality.includes('aggressive') || personality.includes('volatile') || personality.includes('desperate')) {
            return 50 + Math.random() * 25; // 50-75
        }

        return 20 + Math.random() * 30; // Default: 20-50
    }

// Calculate stress increase based on question content
    calculateStressIncrease(question, characterName) {
        const questionLower = question.toLowerCase();
        let stressIncrease = 5; // Base increase for any question

        // Check for high-stress keywords
        for (const keyword of this.stressKeywords.high) {
            if (questionLower.includes(keyword)) {
                stressIncrease += 15;
            }
        }

        // Check for medium-stress keywords
        for (const keyword of this.stressKeywords.medium) {
            if (questionLower.includes(keyword)) {
                stressIncrease += 8;
            }
        }

        // Reduce stress for low-stress keywords
        for (const keyword of this.stressKeywords.low) {
            if (questionLower.includes(keyword)) {
                stressIncrease = Math.max(1, stressIncrease - 5);
            }
        }

        // Add randomness
        stressIncrease += Math.random() * 10 - 5; // ±5 random variation

        // Character-specific modifiers
        const character = this.currentMystery.characters.find(c => c.name === characterName);
        if (character) {
            const personality = character.personality.toLowerCase();
            if (personality.includes('nervous')) stressIncrease *= 1.3;
            if (personality.includes('calm')) stressIncrease *= 0.7;
            if (personality.includes('secretive')) stressIncrease *= 1.2;
        }

        return Math.max(1, Math.round(stressIncrease));
    }

// Apply stress level visual effects to character card
    applyStressLevel(card, stressLevel) {
        // Remove existing stress classes
        card.classList.remove('stress-low', 'stress-medium', 'stress-high', 'stress-extreme');

        // Apply appropriate stress class
        if (stressLevel < 25) {
            card.classList.add('stress-low');
        } else if (stressLevel < 50) {
            card.classList.add('stress-medium');
        } else if (stressLevel < 80) {
            card.classList.add('stress-high');
        } else {
            card.classList.add('stress-extreme');
        }

        // Update stress bars
        this.updateStressBars(card, stressLevel);

        // Update stress label
        const stressLabel = card.querySelector('.stress-label');
        if (stressLabel) {
            stressLabel.textContent = this.getStressLabel(stressLevel);
        }
    }

    // Update stress bar fill levels
    updateStressBars(card, stressLevel) {
        const stressBars = card.querySelectorAll('.stress-fill');
        const fillPercentage = stressLevel / 100;
        const barsToFill = Math.ceil(fillPercentage * 5); // 5 bars total

        stressBars.forEach((bar, index) => {
            if (index < barsToFill) {
                const barFill = Math.min(100, Math.max(0, (fillPercentage * 5 - index) * 100));
                bar.style.height = `${barFill}%`;
            } else {
                bar.style.height = '0%';
            }
        });
    }

    // Get stress level label
    getStressLabel(stressLevel) {
        if (stressLevel < 25) return 'Calm';
        if (stressLevel < 40) return 'Composed';
        if (stressLevel < 55) return 'Nervous';
        if (stressLevel < 70) return 'Agitated';
        if (stressLevel < 85) return 'Stressed';
        return 'Panicking';
    }

    startInvestigation() {
        document.getElementById('intro-section').classList.add('hidden');
        document.getElementById('investigation-section').classList.remove('hidden');
        
        // Enable accuse button
        document.getElementById('accuse-btn').disabled = false;
        
        // Add welcome message to conversation
        this.addMessage('system', 'Investigation Started', 'You may now question the suspects. Select a character from the left panel and ask them questions.');
    }

    selectCharacter(characterName) {
        // Update UI
        document.querySelectorAll('.character-card').forEach(card => {
            card.classList.remove('selected');
        });
        
        document.querySelector(`[data-character-name="${characterName}"]`).classList.add('selected');
        
        // Enable input
        document.getElementById('question-input').disabled = false;
        document.getElementById('ask-btn').disabled = false;
        
        // Show selected character
        document.getElementById('selected-character').classList.remove('hidden');
        document.getElementById('selected-character-name').textContent = characterName;
        
        this.selectedCharacter = characterName;
        
        // Focus on input
        document.getElementById('question-input').focus();
    }

    // Enhanced askQuestion method
    async askQuestion() {
        const questionInput = document.getElementById('question-input');
        const question = questionInput.value.trim();

        if (!question || !this.selectedCharacter) return;

        // Disable input while processing
        questionInput.disabled = true;
        document.getElementById('ask-btn').disabled = true;

        // Calculate stress increase
        const stressIncrease = this.calculateStressIncrease(question, this.selectedCharacter);

        // Add question to conversation
        this.addMessage('detective', 'You', question);

        // Clear input
        questionInput.value = '';

        try {
            const response = await fetch(`/api/v1/game/${this.currentSession}/ask`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    character_name: this.selectedCharacter,
                    question: question,
                    current_stress: this.characterStressLevels[this.selectedCharacter] || 25
                })
            });

            const data = await response.json();

            if (response.ok) {
                // Update character stress
                this.characterStressLevels[this.selectedCharacter] = Math.min(100,
                    (this.characterStressLevels[this.selectedCharacter] || 25) + stressIncrease);

                // Apply visual stress updates
                this.updateCharacterStress(this.selectedCharacter);

                // Add character response with stress indication
                const stressEmoji = this.getStressEmoji(this.characterStressLevels[this.selectedCharacter]);
                this.addMessage('character', `${data.character} ${stressEmoji}`, `${data.response} [${data.emotion}]`);

                // Play TTS audio for character response
                if (this.ttsEnabled) {
                    this.playTTS(data.response, data.character, data.emotion);
                }

                // Check for confession or breakdown
                this.checkForStressEvents(this.selectedCharacter);

            } else {
                throw new Error(data.error || 'Failed to get response');
            }
        } catch (error) {
            console.error('Failed to ask question:', error);
            this.addMessage('system', 'Error', 'Failed to get a response. Please try again.');
        } finally {
            // Re-enable input
            questionInput.disabled = false;
            document.getElementById('ask-btn').disabled = false;
            questionInput.focus();
        }
    }

    // Update character stress visually
    updateCharacterStress(characterName) {
        const characterCard = document.querySelector(`[data-character-name="${characterName}"]`);
        if (characterCard) {
            const stressLevel = this.characterStressLevels[characterName];
            this.applyStressLevel(characterCard, stressLevel);

            // Add temporary flash effect
            characterCard.style.background = 'rgba(255, 193, 7, 0.2)';
            setTimeout(() => {
                characterCard.style.background = '';
            }, 500);
        }
    }

// Get stress emoji for messages
    getStressEmoji(stressLevel) {
        if (stressLevel < 25) return '😌';
        if (stressLevel < 40) return '🙂';
        if (stressLevel < 55) return '😐';
        if (stressLevel < 70) return '😰';
        if (stressLevel < 85) return '😨';
        return '😱';
    }

// Check for special stress events
    checkForStressEvents(characterName) {
        const stressLevel = this.characterStressLevels[characterName];

        if (stressLevel >= 90 && Math.random() < 0.3) {
            // Character might confess or break down
            setTimeout(() => {
                this.triggerStressEvent(characterName, 'confession');
            }, 1000);
        } else if (stressLevel >= 95 && Math.random() < 0.5) {
            // Character refuses to talk further
            setTimeout(() => {
                this.triggerStressEvent(characterName, 'shutdown');
            }, 800);
        }
    }

// Trigger special stress events
    triggerStressEvent(characterName, eventType) {
        const character = this.currentMystery.characters.find(c => c.name === characterName);
        if (!character) return;

        switch (eventType) {
            case 'confession':
                this.addMessage('system', 'CONFESSION!',
                    `${characterName} is under extreme stress and might be ready to confess something important!`);
                break;
            case 'shutdown':
                this.addMessage('system', 'BREAKDOWN',
                    `${characterName} is too stressed to continue and refuses to answer more questions for now.`);
                // Temporarily disable this character
                this.disableCharacterTemporarily(characterName);
                break;
        }
    }

// Temporarily disable a character due to stress
    disableCharacterTemporarily(characterName) {
        const characterCard = document.querySelector(`[data-character-name="${characterName}"]`);
        if (characterCard) {
            characterCard.style.opacity = '0.5';
            characterCard.style.pointerEvents = 'none';

            // Re-enable after 30 seconds with reduced stress
            setTimeout(() => {
                this.characterStressLevels[characterName] = Math.max(30, this.characterStressLevels[characterName] - 40);
                this.updateCharacterStress(characterName);
                characterCard.style.opacity = '1';
                characterCard.style.pointerEvents = 'auto';

                this.addMessage('system', 'Recovery',
                    `${characterName} has calmed down and is willing to talk again.`);
            }, 30000);
        }
    }

    addMessage(type, sender, content) {
        const conversation = document.getElementById('conversation-history');
        
        const message = document.createElement('div');
        message.className = `message ${type}`;

        let messageContent = content;
        if (type === 'character' && !this.hintsEnabled && this.ttsEnabled) {
            messageContent = '<i>(Listen to the audio response)</i>';
        }
        
        message.innerHTML = `
            <div class="message-header">${sender}</div>
            <div class="message-content">${messageContent}</div>
        `;
        
        conversation.appendChild(message);
        
        // Scroll to bottom
        conversation.scrollTop = conversation.scrollHeight;
    }

    showScreen(screenId) {
        // Hide all screens
        document.querySelectorAll('.screen').forEach(screen => {
            screen.classList.remove('active');
        });
        
        // Show target screen
        document.getElementById(screenId).classList.add('active');
    }

    showError(message) {
        alert(message); // Simple error handling for now
    }

    resetGame() {
        // Clear timer
        if (this.timerInterval) {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }

        // Clear game state
        this.currentSession = null;
        this.currentMystery = null;
        this.selectedCharacter = null;
        this.characterStressLevels = {};

        // Clear UI elements
        document.getElementById('conversation-history').innerHTML = '';
        document.getElementById('question-input').value = '';
        document.getElementById('question-input').disabled = true;
        document.getElementById('ask-btn').disabled = true;
        document.getElementById('accuse-btn').disabled = true;
        document.getElementById('selected-character').classList.add('hidden');

        // Clear character selection and stress
        document.querySelectorAll('.character-card').forEach(card => {
            card.classList.remove('selected');
            card.classList.remove('stress-low', 'stress-medium', 'stress-high', 'stress-extreme');
            card.style.opacity = '1';
            card.style.pointerEvents = 'auto';
        });

        // Clear character selection
        document.querySelectorAll('.character-card').forEach(card => {
            card.classList.remove('selected');
        });

        // Hide investigation section
        document.getElementById('investigation-section').classList.add('hidden');
        
        // Show intro section
        document.getElementById('intro-section').classList.remove('hidden');
    }

    async toggleTimer() {
        if (!this.currentSession) return;

        try {
            const response = await fetch(`/api/v1/game/${this.currentSession}/timer/toggle`, {
                method: 'POST'
            });
            const data = await response.json();

            if (response.ok) {
                this.timerEnabled = data.timer_enabled;
                document.getElementById('timer-toggle-btn').textContent = this.timerEnabled ? '⏳ Timer On' : '⏳ Timer Off';
            }
        } catch (error) {
            console.error('Failed to toggle timer:', error);
        }
    }

    async updateTimer() {
        if (!this.currentSession) return;

        try {
            const response = await fetch(`/api/v1/game/${this.currentSession}/timer`);
            const data = await response.json();

            if (response.ok) {
                if (data.game_over) {
                    clearInterval(this.timerInterval);
                    this.showResultModal(false, 'You ran out of time!');
                    return;
                }

                const timerDisplay = document.getElementById('timer');
                const minutes = Math.floor(data.remaining_time / 60);
                const seconds = data.remaining_time % 60;
                timerDisplay.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;

                if (data.remaining_time <= 60 && data.timer_enabled) {
                    timerDisplay.classList.add('warning');
                } else {
                    timerDisplay.classList.remove('warning');
                }
            }
        } catch (error) {
            console.error('Failed to update timer:', error);
            clearInterval(this.timerInterval);
        }
    }

    showAccusationModal() {
        if (!this.currentMystery || !this.currentMystery.characters) {
            this.showError('No characters available for accusation');
            return;
        }

        // Populate character buttons
        const container = document.getElementById('accusation-characters');
        container.innerHTML = '';

        this.currentMystery.characters.forEach(character => {
            const button = document.createElement('button');
            button.className = 'accusation-character-btn';
            button.textContent = character.name;
            button.addEventListener('click', () => {
                this.makeAccusation(character.name);
            });
            container.appendChild(button);
        });

        this.showModal('accusation-modal');
    }

    async makeAccusation(accusedName) {
        this.hideModal('accusation-modal');
        
        // Check if the accusation is correct
        const actualKiller = this.getActualKiller();
        const isCorrect = accusedName === actualKiller;

        this.showResultModal(isCorrect, accusedName);
    }

    getActualKiller() {
        // Get killer from the mystery data
        return this.currentMystery ? this.currentMystery.killer : null;
    }

    showResultModal(isCorrect, accusedName) {
        const title = document.getElementById('result-title');
        const message = document.getElementById('result-message');

        if (isCorrect) {
            title.textContent = '🎉 Congratulations!';
            title.style.color = '#4CAF50';
            message.textContent = `You correctly identified ${accusedName} as the killer! Your detective skills have solved the case.`;
        } else {
            title.textContent = '❌ Case Closed... Incorrectly';
            title.style.color = '#dc3545';
            message.textContent = `Unfortunately, ${accusedName} was not the killer. The real culprit remains free. Better luck next time, detective.`;
        }

        this.showModal('result-modal');
    }

    showModal(modalId) {
        document.getElementById(modalId).classList.remove('hidden');
    }

    hideModal(modalId) {
        document.getElementById(modalId).classList.add('hidden');
    }

    async playTTS(text, character, emotion) {
        try {
            // Stop any currently playing audio
            this.stopTTS();

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
                // Get audio data as blob
                const audioBlob = await response.blob();
                
                // Create audio URL and play
                const audioUrl = URL.createObjectURL(audioBlob);
                this.currentAudio = new Audio(audioUrl);
                
                // Clean up URL when audio ends
                this.currentAudio.addEventListener('ended', () => {
                    URL.revokeObjectURL(audioUrl);
                    this.currentAudio = null;
                });

                // Set volume and try to play the audio
                this.currentAudio.volume = 0.8;
                
                try {
                    await this.currentAudio.play();
                    console.log(`🔊 Playing TTS for ${character}: ${text.substring(0, 50)}...`);
                } catch (playError) {
                    console.error('Audio play failed:', playError);
                    console.log('This might be due to browser autoplay policy. User interaction required.');
                    
                    // Show a notification that user needs to interact
                    this.showAudioPermissionPrompt();
                }
            } else {
                console.warn('TTS failed:', response.statusText);
            }
        } catch (error) {
            console.error('TTS Error:', error);
        }
    }

    stopTTS() {
        if (this.currentAudio) {
            this.currentAudio.pause();
            this.currentAudio = null;
        }
    }

    toggleTTS() {
        this.ttsEnabled = !this.ttsEnabled;
        if (!this.ttsEnabled) {
            this.stopTTS();
        }
        console.log(`TTS ${this.ttsEnabled ? 'enabled' : 'disabled'}`);
        return this.ttsEnabled;
    }

    showAudioPermissionPrompt() {
        // Create a temporary notification
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #d4af37;
            color: #1a1a2e;
            padding: 1rem;
            border-radius: 8px;
            z-index: 1001;
            font-weight: bold;
            cursor: pointer;
        `;
        notification.textContent = '🔊 Click here to enable audio';
        
        notification.addEventListener('click', () => {
            this.testAudioPermission();
            notification.remove();
        });
        
        document.body.appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.remove();
            }
        }, 5000);
    }

    async testAudioPermission() {
        try {
            // Create a short silent audio to unlock browser audio
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const oscillator = audioContext.createOscillator();
            const gainNode = audioContext.createGain();
            
            oscillator.connect(gainNode);
            gainNode.connect(audioContext.destination);
            
            gainNode.gain.setValueAtTime(0, audioContext.currentTime);
            oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
            
            oscillator.start();
            oscillator.stop(audioContext.currentTime + 0.1);
            
            console.log('✅ Audio permission granted');
            
            // Test with actual TTS
            this.playTestTTS();
            
        } catch (error) {
            console.error('Audio permission test failed:', error);
        }
    }

    async playTestTTS() {
        try {
            const response = await fetch('/api/v1/tts/test');
            if (response.ok) {
                const audioBlob = await response.blob();
                const audioUrl = URL.createObjectURL(audioBlob);
                const testAudio = new Audio(audioUrl);
                
                testAudio.volume = 0.8;
                await testAudio.play();
                
                console.log('🔊 TTS test audio played successfully');
                
                testAudio.addEventListener('ended', () => {
                    URL.revokeObjectURL(audioUrl);
                });
            }
        } catch (error) {
            console.error('TTS test failed:', error);
        }
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new GoFigureApp();
});

