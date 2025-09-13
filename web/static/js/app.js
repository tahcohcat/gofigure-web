// web/static/js/app.js - Updated with user features

class MysteryGame {
    constructor() {
        this.currentSession = null;
        this.selectedCharacter = null;
        this.characterStressLevels = {};
        this.ttsEnabled = true;
        this.hintsEnabled = true;
        this.currentUser = null;
        this.userStats = null;

        this.init();
    }

    async init() {
        await this.loadMysteries();
        this.setupEventListeners();
        await this.checkAuthStatus();
    }

    async checkAuthStatus() {
        try {
            const response = await fetch('/api/v1/auth/profile', {
                headers: { 'Accept': 'application/json' }
            });

            if (response.ok) {
                const data = await response.json();
                console.log('Profile data:', data);
                this.currentUser = data.user;
                this.userStats = data.stats;
                this.showUserInfo();
            } else {
                console.log('checkAuthStatus response not ok:', response.status);
            }
        } catch (error) {
            console.log('Not authenticated or error checking auth:', error);
        }
    }

    showUserInfo() {
        console.log('Inside showUserInfo');
        const user = this.currentUser;
        const stats = this.userStats;

        if (!user || !stats) {
            console.log('User or stats is null, returning.');
            return;
        }

        document.getElementById('user-display-name').textContent = user.display_name;
        document.getElementById('stats-games-played').textContent = stats.games_played;
        document.getElementById('stats-games-won').textContent = stats.games_won;
        document.getElementById('stats-play-time').textContent = Math.floor(stats.total_play_time / 60);
        document.getElementById('stats-fastest-solve').textContent = stats.fastest_solve > 0 ? Math.floor(stats.fastest_solve / 60) + 'm' : 'N/A';

        document.querySelector('.user-info').classList.remove('hidden');
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
                <div class="mystery-header">
                    <h3>${mystery.title}</h3>
                    <div class="difficulty-badge difficulty-${mystery.difficulty.toLowerCase()}">${mystery.difficulty}</div>
                </div>
                <p>${mystery.description}</p>
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

// Add these functions to your existing web/static/js/app.js

// Profile-related functionality
const profile = {
    // Show profile page
    show: function() {
        // Create profile modal or navigate to profile page
        const profileModal = this.createProfileModal();
        document.body.appendChild(profileModal);
        this.loadProfileData();
    },

    // Create profile modal HTML
    createProfileModal: function() {
        const modal = document.createElement('div');
        modal.id = 'profile-modal';
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content profile-modal-content">
                <div class="modal-header">
                    <h2>🕵️ Detective Profile</h2>
                    <span class="close-modal" onclick="profile.close()">&times;</span>
                </div>
                
                <div class="profile-content">
                    <div class="profile-info">
                        <div class="profile-avatar">🕵️</div>
                        <h3 id="modal-profile-name">Loading...</h3>
                        <p class="detective-rank" id="modal-detective-rank">🔍 Detective Trainee</p>
                    </div>
                    
                    <div class="profile-stats-grid">
                        <div class="profile-stat">
                            <span class="stat-icon">🎯</span>
                            <div class="stat-value" id="modal-games-played">-</div>
                            <div class="stat-label">Cases</div>
                        </div>
                        <div class="profile-stat">
                            <span class="stat-icon">✅</span>
                            <div class="stat-value" id="modal-games-won">-</div>
                            <div class="stat-label">Solved</div>
                        </div>
                        <div class="profile-stat">
                            <span class="stat-icon">⚡</span>
                            <div class="stat-value" id="modal-success-rate">-</div>
                            <div class="stat-label">Success Rate</div>
                        </div>
                        <div class="profile-stat">
                            <span class="stat-icon">🏅</span>
                            <div class="stat-value" id="modal-badges-earned">-</div>
                            <div class="stat-label">Badges</div>
                        </div>
                    </div>
                    
                    <div class="profile-section">
                        <h4>🏆 Recent Achievements</h4>
                        <div id="modal-achievements" class="achievements-preview">
                            Loading achievements...
                        </div>
                    </div>
                    
                    <div class="profile-section">
                        <h4>📋 Recent Activity</h4>
                        <div id="modal-activities" class="activities-preview">
                            Loading activities...
                        </div>
                    </div>
                    
                    <div class="profile-actions">
                        <button class="btn btn-primary" onclick="profile.openFullProfile()">
                            View Full Profile
                        </button>
                        <button class="btn btn-secondary" onclick="profile.close()">
                            Close
                        </button>
                    </div>
                </div>
            </div>
        `;

        // Add modal styles
        const style = document.createElement('style');
        style.textContent = `
            .profile-modal-content {
                max-width: 600px;
                max-height: 80vh;
                overflow-y: auto;
            }
            
            .profile-info {
                text-align: center;
                margin-bottom: 2rem;
            }
            
            .profile-avatar {
                width: 80px;
                height: 80px;
                border-radius: 50%;
                background: linear-gradient(135deg, #667eea, #764ba2);
                display: flex;
                align-items: center;
                justify-content: center;
                font-size: 2rem;
                margin: 0 auto 1rem;
                color: white;
            }
            
            .detective-rank {
                background: rgba(102, 126, 234, 0.1);
                color: #667eea;
                padding: 0.5rem 1rem;
                border-radius: 20px;
                font-size: 0.9rem;
                font-weight: 600;
                display: inline-block;
            }
            
            .profile-stats-grid {
                display: grid;
                grid-template-columns: repeat(2, 1fr);
                gap: 1rem;
                margin-bottom: 2rem;
            }
            
            .profile-stat {
                background: #f8f9ff;
                padding: 1rem;
                border-radius: 10px;
                text-align: center;
                border: 1px solid #e1e8f7;
            }
            
            .profile-stat .stat-icon {
                font-size: 1.5rem;
                display: block;
                margin-bottom: 0.5rem;
            }
            
            .profile-stat .stat-value {
                font-size: 1.5rem;
                font-weight: 700;
                color: #667eea;
                margin-bottom: 0.25rem;
            }
            
            .profile-stat .stat-label {
                font-size: 0.8rem;
                color: #666;
                text-transform: uppercase;
                letter-spacing: 0.5px;
            }
            
            .profile-section {
                margin-bottom: 1.5rem;
            }
            
            .profile-section h4 {
                margin-bottom: 1rem;
                color: #333;
                border-bottom: 2px solid #e1e8f7;
                padding-bottom: 0.5rem;
            }
            
            .achievements-preview, .activities-preview {
                max-height: 150px;
                overflow-y: auto;
            }
            
            .achievement-mini {
                display: flex;
                align-items: center;
                padding: 0.5rem;
                margin-bottom: 0.5rem;
                background: white;
                border-radius: 8px;
                border: 1px solid #e1e8f7;
            }
            
            .achievement-mini.earned {
                border-color: #4ecdc4;
                background: #f0fffe;
            }
            
            .achievement-mini .badge-icon {
                font-size: 1.2rem;
                margin-right: 0.75rem;
                width: 24px;
                text-align: center;
            }
            
            .achievement-mini .badge-info {
                flex: 1;
            }
            
            .achievement-mini .badge-title {
                font-weight: 600;
                font-size: 0.9rem;
                margin-bottom: 0.2rem;
            }
            
            .achievement-mini .badge-description {
                font-size: 0.8rem;
                color: #666;
            }
            
            .activity-mini {
                display: flex;
                align-items: center;
                padding: 0.5rem 0;
                border-bottom: 1px solid #f0f0f0;
            }
            
            .activity-mini:last-child {
                border-bottom: none;
            }
            
            .activity-mini .activity-icon {
                font-size: 1.2rem;
                margin-right: 0.75rem;
                width: 24px;
                text-align: center;
            }
            
            .activity-mini .activity-text {
                flex: 1;
                font-size: 0.9rem;
            }
            
            .activity-mini .activity-time {
                font-size: 0.8rem;
                color: #666;
            }
            
            .profile-actions {
                display: flex;
                gap: 1rem;
                justify-content: center;
                margin-top: 2rem;
                padding-top: 1rem;
                border-top: 1px solid #e1e8f7;
            }
            
            @media (max-width: 600px) {
                .profile-stats-grid {
                    grid-template-columns: repeat(2, 1fr);
                }
                
                .profile-actions {
                    flex-direction: column;
                }
            }
        `;
        document.head.appendChild(style);

        return modal;
    },

    // Load profile data from API
    loadProfileData: async function() {
        try {
            const response = await fetch('/api/v1/profile/full');
            if (!response.ok) throw new Error('Failed to load profile');

            const data = await response.json();

            // Update profile info
            document.getElementById('modal-profile-name').textContent = data.user.display_name;
            document.getElementById('modal-detective-rank').textContent = data.stats.detective_rank;

            // Update stats
            document.getElementById('modal-games-played').textContent = data.stats.games_played;
            document.getElementById('modal-games-won').textContent = data.stats.games_won;
            document.getElementById('modal-success-rate').textContent = data.stats.success_rate + '%';
            document.getElementById('modal-badges-earned').textContent = data.stats.badges_earned;

            // Update achievements
            this.renderAchievements(data.achievements.slice(0, 5)); // Show first 5

            // Update activities
            this.renderActivities(data.activities.slice(0, 5)); // Show first 5

        } catch (error) {
            console.error('Error loading profile:', error);
            document.getElementById('modal-profile-name').textContent = 'Error loading profile';
        }
    },

    // Render achievements preview
    renderAchievements: function(achievements) {
        const container = document.getElementById('modal-achievements');

        if (!achievements || achievements.length === 0) {
            container.innerHTML = '<p>No achievements yet. Keep investigating!</p>';
            return;
        }

        container.innerHTML = achievements.map(achievement => `
            <div class="achievement-mini ${achievement.completed ? 'earned' : ''}">
                <span class="badge-icon" style="${achievement.completed ? '' : 'filter: grayscale(100%); opacity: 0.6;'}">${achievement.icon}</span>
                <div class="badge-info">
                    <div class="badge-title">${achievement.title}</div>
                    <div class="badge-description">${achievement.description}</div>
                </div>
            </div>
        `).join('');
    },

    // Render activities preview
    renderActivities: function(activities) {
        const container = document.getElementById('modal-activities');

        if (!activities || activities.length === 0) {
            container.innerHTML = '<p>No recent activity.</p>';
            return;
        }

        container.innerHTML = activities.map(activity => `
            <div class="activity-mini">
                <span class="activity-icon">${activity.icon}</span>
                <span class="activity-text">${activity.text}</span>
                <span class="activity-time">${activity.time}</span>
            </div>
        `).join('');
    },

    // Open full profile page in new tab
    openFullProfile: function() {
        // Save the profile page HTML as a separate route or open in new window
        const profileWindow = window.open('', '_blank');
        profileWindow.document.write(this.getFullProfileHTML());
        profileWindow.document.close();

        // Load the profile data in the new window
        profileWindow.addEventListener('DOMContentLoaded', () => {
            profileWindow.loadProfileData();
        });
    },

    // Get full profile HTML (you could also make this a separate page)
    getFullProfileHTML: function() {
        // Return the HTML from your profile artifact
        // This is a simplified version - in practice, you'd serve this as a separate page
        return `
            <!DOCTYPE html>
            <html>
            <head>
                <title>Detective Profile</title>
                <style>
                    /* Include the CSS from your profile artifact */
                </style>
            </head>
            <body>
                <!-- Include the HTML structure from your profile artifact -->
                <script>
                    // Include the JavaScript from your profile artifact
                </script>
            </body>
            </html>
        `;
    },

    // Close profile modal
    close: function() {
        const modal = document.getElementById('profile-modal');
        if (modal) {
            modal.remove();
        }
    }
};

// Achievement notification system
const achievements = {
    // Show achievement earned notification
    showEarned: function(achievement) {
        const notification = document.createElement('div');
        notification.className = 'achievement-notification';
        notification.innerHTML = `
            <div class="achievement-content">
                <div class="achievement-icon">${achievement.icon}</div>
                <div class="achievement-text">
                    <div class="achievement-title">Achievement Unlocked!</div>
                    <div class="achievement-name">${achievement.title}</div>
                    <div class="achievement-desc">${achievement.description}</div>
                </div>
            </div>
        `;

        // Add notification styles if not already added
        if (!document.getElementById('achievement-styles')) {
            const style = document.createElement('style');
            style.id = 'achievement-styles';
            style.textContent = `
                .achievement-notification {
                    position: fixed;
                    top: 20px;
                    right: 20px;
                    background: linear-gradient(135deg, #4ecdc4, #44a08d);
                    color: white;
                    padding: 1rem;
                    border-radius: 10px;
                    box-shadow: 0 10px 30px rgba(0,0,0,0.3);
                    z-index: 10000;
                    max-width: 350px;
                    animation: slideInRight 0.5s ease-out, fadeOut 0.5s ease-in 4.5s forwards;
                }
                
                .achievement-content {
                    display: flex;
                    align-items: center;
                }
                
                .achievement-icon {
                    font-size: 2rem;
                    margin-right: 1rem;
                }
                
                .achievement-title {
                    font-weight: 700;
                    font-size: 0.9rem;
                    margin-bottom: 0.25rem;
                    text-transform: uppercase;
                    letter-spacing: 0.5px;
                }
                
                .achievement-name {
                    font-weight: 600;
                    font-size: 1.1rem;
                    margin-bottom: 0.25rem;
                }
                
                .achievement-desc {
                    font-size: 0.85rem;
                    opacity: 0.9;
                }
                
                @keyframes slideInRight {
                    from {
                        opacity: 0;
                        transform: translateX(100%);
                    }
                    to {
                        opacity: 1;
                        transform: translateX(0);
                    }
                }
                
                @keyframes fadeOut {
                    to {
                        opacity: 0;
                        transform: translateX(100%);
                    }
                }
            `;
            document.head.appendChild(style);
        }

        document.body.appendChild(notification);

        // Remove notification after animation
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    },

    // Check for new achievements after game events
    checkAfterGameEnd: async function(gameResult) {
        try {
            const response = await fetch('/api/v1/profile/achievements');
            if (!response.ok) return;

            const data = await response.json();
            const recentAchievements = data.achievements.filter(a =>
                a.completed && a.completed_at &&
                new Date(a.completed_at) > new Date(Date.now() - 10000) // Last 10 seconds
            );

            recentAchievements.forEach(achievement => {
                setTimeout(() => this.showEarned(achievement), 1000);
            });

        } catch (error) {
            console.error('Error checking achievements:', error);
        }
    }
};

// Update your existing game object to include profile functionality
if (window.game) {
    // Add profile button to user info dropdown
    game.showProfile = function() {
        profile.show();
    };

    // Update user info display to include profile link
    game.updateUserInfo = function(userData) {
        // Your existing user info update code...

        // Add profile link to dropdown if it doesn't exist
        const dropdown = document.querySelector('.dropdown-content');
        if (dropdown && !dropdown.querySelector('.profile-link')) {
            const profileLink = document.createElement('a');
            profileLink.href = '#';
            profileLink.className = 'profile-link';
            profileLink.textContent = 'View Profile';
            profileLink.onclick = (e) => {
                e.preventDefault();
                game.showProfile();
            };

            // Insert before logout link
            const logoutLink = dropdown.querySelector('a[onclick*="logout"]');
            if (logoutLink) {
                dropdown.insertBefore(profileLink, logoutLink);
            } else {
                dropdown.appendChild(profileLink);
            }
        }
    };

    // Hook into existing game end logic
    const originalEndGame = game.endGame || function() {};
    game.endGame = function(result) {
        originalEndGame.call(this, result);

        // Check for new achievements after a brief delay
        setTimeout(() => {
            achievements.checkAfterGameEnd(result);
        }, 2000);
    };
}

// Add keyboard shortcut for profile (P key)
document.addEventListener('keydown', function(e) {
    if (e.key === 'p' || e.key === 'P') {
        if (!document.querySelector('.modal') && window.game) {
            game.showProfile();
        }
    }
});

// Add this to your existing web/static/js/app.js

// Achievement system integration for real-time notifications
async function handleGameCompletion(result) {
    try {
        // Your existing game completion logic...

        // After a brief delay, check for new achievements
        setTimeout(async () => {
            const response = await fetch('/api/v1/profile/achievements');
            if (response.ok) {
                const data = await response.json();

                // Find recently earned achievements (within last 30 seconds)
                const recentAchievements = data.achievements.filter(achievement => {
                    if (!achievement.completed || !achievement.completed_at) return false;

                    const completedTime = new Date(achievement.completed_at);
                    const now = new Date();
                    const timeDiff = now - completedTime;

                    return timeDiff < 30000; // 30 seconds
                });

                // Show notifications for new achievements
                recentAchievements.forEach((achievement, index) => {
                    setTimeout(() => {
                        showAchievementNotification(achievement);
                    }, index * 1000); // Stagger notifications
                });
            }
        }, 2000);

    } catch (error) {
        console.error('Error checking achievements:', error);
    }
}

function showAchievementNotification(achievement) {
    const notification = document.createElement('div');
    notification.className = 'achievement-toast';
    notification.innerHTML = `
        <div style="display: flex; align-items: center;">
            <div style="font-size: 2rem; margin-right: 1rem;">${achievement.icon}</div>
            <div>
                <div style="font-weight: 700; margin-bottom: 0.25rem;">🎉 Achievement Unlocked!</div>
                <div style="font-weight: 600; margin-bottom: 0.25rem;">${achievement.title}</div>
                <div style="font-size: 0.9rem; opacity: 0.9;">${achievement.description}</div>
            </div>
        </div>
    `;

    notification.onclick = () => {
        notification.remove();
        // Optionally open profile to show all achievements
        if (window.profile) {
            profile.show();
        }
    };

    document.body.appendChild(notification);

    // Play achievement sound (optional)
    playAchievementSound();

    // Remove after 5 seconds
    setTimeout(() => {
        if (notification.parentNode) {
            notification.remove();
        }
    }, 5000);
}

function playAchievementSound() {
    // Create a simple achievement sound using Web Audio API
    try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();

        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);

        oscillator.frequency.setValueAtTime(800, audioContext.currentTime);
        oscillator.frequency.exponentialRampToValueAtTime(1200, audioContext.currentTime + 0.1);
        oscillator.frequency.exponentialRampToValueAtTime(900, audioContext.currentTime + 0.3);

        gainNode.gain.setValueAtTime(0.1, audioContext.currentTime);
        gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3);

        oscillator.start(audioContext.currentTime);
        oscillator.stop(audioContext.currentTime + 0.3);
    } catch (error) {
        // Fallback or ignore if Web Audio API is not supported
        console.log('Achievement sound not available');
    }
}

// Progress tracking functions
function updateAchievementProgress(achievementId, progress) {
    // Update UI elements that show progress
    const progressBars = document.querySelectorAll(`[data-achievement="${achievementId}"] .progress-fill`);
    progressBars.forEach(bar => {
        const maxProgress = parseInt(bar.parentElement.dataset.max) || 100;
        const percentage = (progress / maxProgress) * 100;
        bar.style.width = `${Math.min(percentage, 100)}%`;
    });
}

function trackQuestionAsked() {
    // Call this each time a question is asked
    fetch('/api/v1/profile/achievements')
        .then(response => response.json())
        .then(data => {
            const interrogatorAchievement = data.achievements.find(a => a.id === 'interrogator');
            if (interrogatorAchievement && !interrogatorAchievement.completed) {
                updateAchievementProgress('interrogator', interrogatorAchievement.progress);
            }
        })
        .catch(error => console.error('Error updating question progress:', error));
}

function trackMysteryProgress() {
    // Update mystery completion progress
    fetch('/api/v1/profile/achievements')
        .then(response => response.json())
        .then(data => {
            const mysteryMavenAchievement = data.achievements.find(a => a.id === 'mystery-maven');
            if (mysteryMavenAchievement && !mysteryMavenAchievement.completed) {
                updateAchievementProgress('mystery-maven', mysteryMavenAchievement.progress);
            }
        })
        .catch(error => console.error('Error updating mystery progress:', error));
}

// Hook these functions into your existing game logic
if (window.game) {
    // Update your existing game.askQuestion function to call trackQuestionAsked()
    const originalAskQuestion = game.askQuestion || function() {};
    game.askQuestion = function(...args) {
        const result = originalAskQuestion.apply(this, args);
        trackQuestionAsked();
        return result;
    };

    // Update your existing game end function to call handleGameCompletion()
    const originalEndGame = game.endGame || function() {};
    game.endGame = function(result) {
        const gameResult = originalEndGame.call(this, result);
        handleGameCompletion(result);
        return gameResult;
    };
}

// Initialize game when page loads
let game;
document.addEventListener('DOMContentLoaded', () => {
    game = new MysteryGame();
});