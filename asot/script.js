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
            resultElement.innerHTML = `<p>${result}</p>`;
            searchResultsContainer.appendChild(resultElement);
        });
    }

    function displayResultsCount(count) {
        searchCountContainer.textContent = `
        Found ${count} results`;
    }

    function displayError(message) {
        searchResultsContainer.innerHTML = `<div class="result"><p>${message}</p></div>`;
        searchCountContainer.textContent = '';
    }
});
