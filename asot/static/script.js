document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('searchForm');
    const searchInput = document.getElementById('searchQuery');
    const searchResultsContainer = document.getElementById('searchResults');
    const searchCountContainer = document.getElementById('searchCount');

    // Check if play mode is enabled
    const urlParams = new URLSearchParams(window.location.search);
    const playModeEnabled = urlParams.get('play') === 'true';

    // Audio state management
    let currentAudio = null;
    let currentPlayButton = null;
    let currentEpisodeNumber = null;
    let currentProgressPanel = null;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const query = searchInput.value.trim();

        if (query === '') {
            return;
        }

        try {
            const response = await fetch(`search?query=${encodeURIComponent(query)}`, {
                method: 'POST'
            });

            const data = await response.json();

            if (data && Array.isArray(data.results)) {
                displayResults(data.results);
                displayResultsCount(data.count);
            } else {
                displayError('No results found.');
                displayResultsCount(0);
            }
        } catch (error) {
            console.error('Error fetching search results:', error);
            displayError('An error occurred. Please try again.');
            displayResultsCount(0);
        }
    });

    function displayResults(results) {
        searchResultsContainer.innerHTML = '';

        results.forEach((result) => {
            const resultElement = document.createElement('div');
            resultElement.classList.add('result');
            
            // Create the main content wrapper
            const contentWrapper = document.createElement('div');
            contentWrapper.classList.add('result-content');
            
            const titleElement = document.createElement('p');
            titleElement.textContent = result.title;
            contentWrapper.appendChild(titleElement);
            
            // Add play button if play mode is enabled
            if (playModeEnabled) {
                const episodeNumber = extractEpisodeNumber(result.title);
                if (episodeNumber) {
                    const playControlsWrapper = document.createElement('div');
                    playControlsWrapper.classList.add('play-controls-wrapper');
                    
                    // Create progress panel (hidden by default)
                    const progressPanel = document.createElement('div');
                    progressPanel.classList.add('progress-panel');
                    progressPanel.innerHTML = `
                        <div class="mini-progress-container">
                            <div class="mini-progress-bg">
                                <div class="mini-progress-fill"></div>
                                <div class="mini-progress-handle"></div>
                            </div>
                        </div>
                        <div class="mini-player-time">
                            <span class="mini-current-time">0:00</span>
                            <span class="mini-time-separator">/</span>
                            <span class="mini-total-time">0:00</span>
                        </div>
                    `;
                    playControlsWrapper.appendChild(progressPanel);
                    
                    // Create error panel (hidden by default)
                    const errorPanel = document.createElement('div');
                    errorPanel.classList.add('error-panel');
                    errorPanel.innerHTML = `
                        <div class="error-message">Failed to load audio</div>
                    `;
                    playControlsWrapper.appendChild(errorPanel);
                    
                    // Create play button wrapper
                    const playButtonWrapper = document.createElement('div');
                    playButtonWrapper.classList.add('play-button-wrapper');
                    
                    const playButton = document.createElement('button');
                    playButton.classList.add('play-button');
                    playButton.innerHTML = `
                        <svg class="play-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                            <path d="M8 5v14l11-7z"/>
                        </svg>
                        <svg class="pause-icon" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" style="display: none;">
                            <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z"/>
                        </svg>
                    `;
                    playButton.title = `Play ASOT ${episodeNumber}`;
                    
                    // Add progress circle
                    const progressCircle = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
                    progressCircle.classList.add('progress-circle');
                    progressCircle.setAttribute('viewBox', '0 0 36 36');
                    progressCircle.innerHTML = `
                        <circle cx="18" cy="18" r="16" fill="none" stroke="#e0e0e0" stroke-width="2"/>
                        <circle cx="18" cy="18" r="16" fill="none" stroke="#007bff" stroke-width="2" 
                                stroke-dasharray="100 100" stroke-dashoffset="100" 
                                transform="rotate(-90 18 18)" class="progress-ring"/>
                    `;
                    playButtonWrapper.appendChild(progressCircle);
                    playButtonWrapper.appendChild(playButton);
                    playControlsWrapper.appendChild(playButtonWrapper);
                    
                    playButton.addEventListener('click', (e) => {
                        e.stopPropagation();
                        toggleAudioPlayback(episodeNumber, playButton, playButtonWrapper, progressPanel, errorPanel);
                    });
                    
                    contentWrapper.appendChild(playControlsWrapper);
                }
            }
            
            resultElement.appendChild(contentWrapper);

            resultElement.addEventListener('click', async () => {
                if (window.getSelection().toString().length) {
                    return
                }
                if (!resultElement.dataset.loaded) {
                    const tracklist = await fetchTracklist(result.episodeHash);
                    const tracklistElement = document.createElement('div');
                    tracklistElement.classList.add('tracklist');
                    tracklistElement.innerHTML = `<p>${tracklist.replace(/\r?\n/g, '<br>')}</p>`;
                    resultElement.appendChild(tracklistElement);
                    resultElement.dataset.loaded = true;
                    toggleTracklist(tracklistElement);
                } else {
                    const tracklistElement = resultElement.querySelector('.tracklist');
                    toggleTracklist(tracklistElement);
                }
            });

            searchResultsContainer.appendChild(resultElement);
        });
    }

    function extractEpisodeNumber(title) {
        const match = title.match(/ASOT\s+(\d+)/i);
        return match ? match[1] : null;
    }

    function toggleAudioPlayback(episodeNumber, button, wrapper, progressPanel, errorPanel) {
        const audioPath = `/asot/audio?episode=${episodeNumber}`;
        
        // Hide error panel if it was shown before
        hideErrorPanel(errorPanel);
        
        // If there's a different audio playing, stop it
        if (currentAudio && currentPlayButton && currentPlayButton !== button) {
            currentAudio.pause();
            updatePlayButtonState(currentPlayButton, false);
            hideProgressPanel(currentProgressPanel);
        }
        
        // If clicking the same button
        if (currentAudio && currentPlayButton === button) {
            if (currentAudio.paused) {
                currentAudio.play();
                updatePlayButtonState(button, true);
            } else {
                currentAudio.pause();
                updatePlayButtonState(button, false);
            }
            return;
        }
        
        // Create new audio element
        const audio = new Audio(audioPath);
        currentAudio = audio;
        currentPlayButton = button;
        currentEpisodeNumber = episodeNumber;
        currentProgressPanel = progressPanel;
        
        // Update progress ring
        const progressRing = wrapper.querySelector('.progress-ring');
        const miniCurrentTime = progressPanel.querySelector('.mini-current-time');
        const miniTotalTime = progressPanel.querySelector('.mini-total-time');
        const miniProgressFill = progressPanel.querySelector('.mini-progress-fill');
        const miniProgressHandle = progressPanel.querySelector('.mini-progress-handle');
        
        audio.addEventListener('timeupdate', () => {
            if (audio.duration) {
                const progress = (audio.currentTime / audio.duration) * 100;
                const offset = 100 - progress;
                progressRing.style.strokeDashoffset = offset;
                
                // Update mini player progress
                miniCurrentTime.textContent = formatTime(audio.currentTime);
                miniProgressFill.style.width = `${progress}%`;
                miniProgressHandle.style.left = `${progress}%`;
            }
        });
        
        // Handle when metadata is loaded (file found)
        audio.addEventListener('loadedmetadata', () => {
            miniTotalTime.textContent = formatTime(audio.duration);
            showProgressPanel(progressPanel);
            setupMiniProgressBar(progressPanel, audio);
        });
        
        // Handle audio end
        audio.addEventListener('ended', () => {
            updatePlayButtonState(button, false);
            progressRing.style.strokeDashoffset = '100';
            hideProgressPanel(progressPanel);
            // Reset state so next click will start fresh
            currentAudio = null;
            currentPlayButton = null;
            currentEpisodeNumber = null;
            currentProgressPanel = null;
        });
        
        // Handle errors (404, etc)
        audio.addEventListener('error', (e) => {
            console.error('Error loading audio:', e);
            updatePlayButtonState(button, false);
            progressRing.style.strokeDashoffset = '100';
            // Show error panel instead of alert
            showErrorPanel(errorPanel);
            // Reset state so next click will try again
            currentAudio = null;
            currentPlayButton = null;
            currentEpisodeNumber = null;
            currentProgressPanel = null;
        });
        
        audio.play().catch((err) => {
            console.error('Error playing audio:', err);
            updatePlayButtonState(button, false);
            progressRing.style.strokeDashoffset = '100';
            showErrorPanel(errorPanel);
            // Reset state
            currentAudio = null;
            currentPlayButton = null;
            currentEpisodeNumber = null;
            currentProgressPanel = null;
        });
        updatePlayButtonState(button, true);
    }

    function showProgressPanel(panel) {
        panel.classList.add('visible');
    }

    function hideProgressPanel(panel) {
        if (panel) {
            panel.classList.remove('visible');
        }
    }

    function showErrorPanel(panel) {
        panel.classList.add('visible');
        // Auto-hide after 3 seconds
        setTimeout(() => {
            hideErrorPanel(panel);
        }, 3000);
    }

    function hideErrorPanel(panel) {
        if (panel) {
            panel.classList.remove('visible');
        }
    }

    function formatTime(seconds) {
        if (!seconds || isNaN(seconds)) return '0:00';
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }

    function setupMiniProgressBar(panel, audio) {
        const container = panel.querySelector('.mini-progress-container');
        const fill = panel.querySelector('.mini-progress-fill');
        const handle = panel.querySelector('.mini-progress-handle');
        let isDragging = false;

        const seek = (e) => {
            if (!audio || !audio.duration) return;
            
            const rect = container.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const percentage = Math.max(0, Math.min(1, x / rect.width));
            const newTime = percentage * audio.duration;
            
            audio.currentTime = newTime;
            fill.style.width = `${percentage * 100}%`;
            handle.style.left = `${percentage * 100}%`;
        };

        container.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            isDragging = true;
            seek(e);
        });

        document.addEventListener('mousemove', (e) => {
            if (isDragging) {
                seek(e);
            }
        });

        document.addEventListener('mouseup', () => {
            isDragging = false;
        });

        container.addEventListener('click', (e) => {
            e.stopPropagation();
            seek(e);
        });
    }

    function updatePlayButtonState(button, isPlaying) {
        const playIcon = button.querySelector('.play-icon');
        const pauseIcon = button.querySelector('.pause-icon');
        
        if (isPlaying) {
            playIcon.style.display = 'none';
            pauseIcon.style.display = 'block';
            button.classList.add('playing');
        } else {
            playIcon.style.display = 'block';
            pauseIcon.style.display = 'none';
            button.classList.remove('playing');
        }
    }

    async function fetchTracklist(episodeHash) {
        try {
            const response = await fetch(`tracklist?hash=${encodeURIComponent(episodeHash)}`);
            return await response.text();
        } catch (error) {
            console.error('Error fetching tracklist:', error);
            return 'Tracklist not available';
        }
    }

    function toggleTracklist(element) {
        element.classList.toggle('show');
    }

    function displayResultsCount(count) {
        searchCountContainer.textContent = `Found ${count} result(s)`;
    }

    function displayError(message) {
        searchResultsContainer.innerHTML = `<div class="result"><p>${message}</p></div>`;
        searchCountContainer.textContent = '';
    }
});
