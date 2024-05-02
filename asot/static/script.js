document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('searchForm');
    const searchInput = document.getElementById('searchQuery');
    const searchResultsContainer = document.getElementById('searchResults');
    const searchCountContainer = document.getElementById('searchCount');

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
            resultElement.innerHTML = `<p>${result.title}</p>`;

            resultElement.addEventListener('click', async () => {
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
