(function () {

    const monthGamesEl = document.querySelector('.monthGames');
    const loaderEl = document.querySelector('.loader');

    // get the monthGames from API
    const getMonthGames = async (year, month) => {
        const API_URL = `https://chess-ajc.piposplace.com/monthgames?year=${year}&month=${month}`;
        const response = await fetch(API_URL);
        // handle 404
        if (!response.ok) {
            throw new Error(`An error occurred: ${response.status}`);
        }
        return await response.json();
    }

    // show the monthGames
    const showMonthGames = (html) => {
        const monthGameEl = document.createElement('blockMonthGame');
        monthGameEl.classList.add('monthGame');

        monthGameEl.innerHTML = html;
        monthGamesEl.appendChild(monthGameEl);

    };

    const hideLoader = () => {
        loaderEl.classList.remove('show');
    };

    const showLoader = () => {
        loaderEl.classList.add('show');
    };

    const hasMoreMonthGames = (year, month) => {
        return year > 0 && month > 0;
    };

    // load monthGames
    const loadMonthGames = async (year, month) => {

        // show the loader
        showLoader();

        // 0.5 second later
        setTimeout(async () => {
            try {
                // if having more monthGames to fetch
                if (hasMoreMonthGames(year, month)) {
                    // call the API to get monthGames
                    const response = await getMonthGames(year, month);
                    // show monthGames
                    showMonthGames(response.html);
                    // update the year and month
                    next_year = response.next_year;
                    next_month = response.next_month;
                }
            } catch (error) {
                console.log(error.message);
            } finally {
                hideLoader();
                gettingMore = false;
            }
        }, 500);

    };

    // control variables
    var d = new Date();
    var next_year = d.getFullYear();
    var next_month = d.getMonth() + 1;
    var gettingMore = true


    window.addEventListener('scroll', () => {
        const {
            scrollTop,
            scrollHeight,
            clientHeight
        } = document.documentElement;

        if (scrollTop + clientHeight >= scrollHeight - 5 &&
            hasMoreMonthGames(next_year, next_month) && gettingMore == false) {
            gettingMore = true;
            loadMonthGames(next_year, next_month);
        }
    }, {
        passive: true
    });

    // initialize
    loadMonthGames(next_year, next_month);

})();